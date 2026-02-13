package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"
	"ForecastSync/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ChainBetEvent 表示从链上解析出来的一次下注事件（由监听模块调用）
// 这里不直接依赖 go-ethereum，方便你后续自由选择监听实现方式。
type ChainBetEvent struct {
	UserWallet string  // 用户钱包地址
	EventUUID  string  // 对应 events.event_uuid
	BetOption  string  // 下注方向（建议与 event_odds.option_name 对齐）
	BetAmount  float64 // 下注金额（USDC 等），链上监听处负责单位转换

	TxHash      string // 链上交易哈希
	BlockNumber int64  // 区块高度

	// RawData 原始事件 JSON（方便排查问题）
	RawData map[string]interface{}
}

// OrderService 负责从链上事件生成聚合订单
type OrderService struct {
	db              *gorm.DB
	logger          *logrus.Logger
	marketRepo      repository.MarketRepository
	orderRepo       repository.OrderRepository
	contractEvents  repository.ContractEventRepository
	tradingAdapters map[uint64]interfaces.TradingAdapter // platformID -> adapter，可为 nil
}

// NewOrderService 创建 OrderService。tradingAdapters 可为 nil，则不调用真实下单
func NewOrderService(db *gorm.DB, logger *logrus.Logger, tradingAdapters map[uint64]interfaces.TradingAdapter) *OrderService {
	return &OrderService{
		db:              db,
		logger:          logger,
		marketRepo:      repository.NewMarketRepository(db),
		orderRepo:       repository.NewOrderRepository(db),
		contractEvents:  repository.NewContractEventRepository(db),
		tradingAdapters: tradingAdapters,
	}
}

// CreateOrderFromChainEvent 处理一条合约下注事件：
// 1. 记录到 contract_events 表（幂等：tx_hash 唯一）
// 2. 查询该赛事在多平台的赔率，按 BetOption 选择最高价格的平台
// 3. 生成一条本地订单，锁定当时的赔率
func (s *OrderService) CreateOrderFromChainEvent(ctx context.Context, ev *ChainBetEvent) error {
	if ev == nil {
		return fmt.Errorf("chain bet event is nil")
	}

	// 1. 记录合约事件（如果 tx_hash 已存在则视为已处理）
	if err := s.saveContractEvent(ctx, ev); err != nil {
		// 对重复事件直接忽略
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			s.logger.WithField("tx_hash", ev.TxHash).Info("重复的链上事件，忽略处理")
			return nil
		}
		return err
	}

	// 2. 根据 EventUUID 找到内部事件
	event, err := s.marketRepo.GetEventByUUID(ctx, ev.EventUUID)
	if err != nil {
		return fmt.Errorf("查询事件失败 event_uuid=%s: %w", ev.EventUUID, err)
	}

	// 3. 查询该事件的所有赔率
	odds, err := s.marketRepo.GetOddsByEventID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("查询事件赔率失败 event_id=%d: %w", event.ID, err)
	}
	if len(odds) == 0 {
		return fmt.Errorf("事件%d没有可用赔率记录", event.ID)
	}

	// 4. 在符合 BetOption 的赔率中选择最高价格的平台
	bestPlatformID, bestPrice, bestOptionName, err := pickBestOdds(odds, ev.BetOption)
	if err != nil {
		return err
	}

	// 5. 生成本地订单，先落库再调用 TradingAdapter 真实下单
	orderUUID := uuid.NewString()
	now := time.Now()
	order := &model.Order{
		OrderUUID:  orderUUID,
		UserWallet: ev.UserWallet,
		EventID:    event.ID,
		PlatformID: bestPlatformID,
		BetOption:  bestOptionName,
		BetAmount:  ev.BetAmount,
		LockedOdds: bestPrice,
		Status:     "pending_place",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.orderRepo.CreateOrder(ctx, order); err != nil {
		return fmt.Errorf("创建订单失败: %w", err)
	}

	if err := s.contractEvents.UpdateOrderUUIDAndProcessed(ctx, ev.TxHash, orderUUID); err != nil {
		s.logger.WithError(err).WithField("tx_hash", ev.TxHash).Warn("回写 contract_events.order_uuid 失败")
	}

	// 6. 若有 TradingAdapter，调用平台下单并回写 platform_order_id 与 status=placed
	if s.tradingAdapters != nil {
		if adapter := s.tradingAdapters[bestPlatformID]; adapter != nil {
			req := &interfaces.PlaceOrderRequest{
				PlatformID:      bestPlatformID,
				PlatformEventID: event.PlatformEventID,
				BetOption:       bestOptionName,
				BetAmount:       ev.BetAmount,
				LockedOdds:      bestPrice,
			}
			platformOrderID, err := adapter.PlaceOrder(ctx, req)
			if err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"order_uuid":  orderUUID,
					"platform_id": bestPlatformID,
				}).Warn("平台下单失败，订单保持 pending_place")
			} else {
				_ = s.orderRepo.UpdatePlatformOrderIDAndStatus(ctx, orderUUID, platformOrderID, "placed")
				s.logger.WithField("order_uuid", orderUUID).WithField("platform_order_id", platformOrderID).Info("平台下单成功")
			}
		} else {
			_ = s.orderRepo.UpdatePlatformOrderIDAndStatus(ctx, orderUUID, "", "placed")
		}
	} else {
		_ = s.orderRepo.UpdatePlatformOrderIDAndStatus(ctx, orderUUID, "", "placed")
	}

	s.logger.WithFields(logrus.Fields{
		"order_uuid":  orderUUID,
		"event_uuid":  ev.EventUUID,
		"platform_id": bestPlatformID,
		"bet_option":  bestOptionName,
		"price":       bestPrice,
	}).Info("链上下注事件已生成本地订单并锁定赔率")

	return nil
}

// saveContractEvent 将链上事件写入 contract_events 表
func (s *OrderService) saveContractEvent(ctx context.Context, ev *ChainBetEvent) error {
	rawBytes, err := json.Marshal(ev.RawData)
	if err != nil {
		return fmt.Errorf("序列化 RawData 失败: %w", err)
	}

	blockNumber := ev.BlockNumber
	now := time.Now()
	ce := &model.ContractEvent{
		EventType:   "BetPlaced",
		OrderUUID:   nil, // 可空，创建订单后回写
		UserWallet:  ev.UserWallet,
		TxHash:      ev.TxHash,
		BlockNumber: &blockNumber,
		EventData:   rawBytes,
		Processed:   false,
		CreatedAt:   now,
	}

	return s.contractEvents.SaveContractEvent(ctx, ce)
}

// pickBestOdds 在所有赔率中挑选 BetOption 对应的最高价格
// 若无法找到匹配 BetOption 的记录，则报错。
func pickBestOdds(odds []*model.EventOdds, betOption string) (platformID uint64, price float64, optionName string, err error) {
	if betOption == "" {
		return 0, 0, "", fmt.Errorf("betOption 不能为空")
	}

	var (
		found bool
		best  float64
		pid   uint64
		name  string
	)

	for _, o := range odds {
		if o.OptionName != betOption {
			continue
		}
		if !found || o.Price > best {
			found = true
			best = o.Price
			pid = o.PlatformID
			name = o.OptionName
		}
	}

	if !found {
		return 0, 0, "", fmt.Errorf("未找到匹配下注方向的赔率: bet_option=%s", betOption)
	}

	return pid, best, name, nil
}

// OrderListItem 订单列表项（含赛事标题）
type OrderListItem struct {
	OrderUUID       string  `json:"order_uuid"`
	UserWallet      string  `json:"user_wallet"`
	EventTitle      string  `json:"event_title"`
	EventID         uint64  `json:"event_id"`
	PlatformID      uint64  `json:"platform_id"`
	PlatformOrderID string  `json:"platform_order_id,omitempty"`
	BetOption       string  `json:"bet_option"`
	BetAmount       float64 `json:"bet_amount"`
	LockedOdds      float64 `json:"locked_odds"`
	Status          string  `json:"status"`
	CreatedAt       int64   `json:"created_at"`
}

// OrderListResult 订单列表返回
type OrderListResult struct {
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
	Items    []OrderListItem `json:"items"`
}

// ListByUser 按用户钱包分页查询订单列表
func (s *OrderService) ListByUser(ctx context.Context, userWallet string, page, pageSize int) (*OrderListResult, error) {
	orders, total, err := s.orderRepo.ListByUser(ctx, userWallet, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]OrderListItem, 0, len(orders))
	for _, o := range orders {
		eventTitle := ""
		if e, err := s.marketRepo.GetEventByID(ctx, o.EventID); err == nil && e != nil {
			eventTitle = e.Title
		}
		po := ""
		if o.PlatformOrderID != nil {
			po = *o.PlatformOrderID
		}
		items = append(items, OrderListItem{
			OrderUUID:       o.OrderUUID,
			UserWallet:      o.UserWallet,
			EventTitle:      eventTitle,
			EventID:         o.EventID,
			PlatformID:      o.PlatformID,
			PlatformOrderID: po,
			BetOption:       o.BetOption,
			BetAmount:       o.BetAmount,
			LockedOdds:      o.LockedOdds,
			Status:          o.Status,
			CreatedAt:       o.CreatedAt.UnixMilli(),
		})
	}
	return &OrderListResult{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Items:    items,
	}, nil
}

// OrderDetail 订单详情（含关联 event 与平台信息）
type OrderDetail struct {
	OrderUUID        string  `json:"order_uuid"`
	UserWallet       string  `json:"user_wallet"`
	EventID          uint64  `json:"event_id"`
	EventUUID        string  `json:"event_uuid"`
	EventTitle       string  `json:"event_title"`
	PlatformID       uint64  `json:"platform_id"`
	PlatformOrderID  string  `json:"platform_order_id,omitempty"`
	BetOption        string  `json:"bet_option"`
	BetAmount        float64 `json:"bet_amount"`
	LockedOdds       float64 `json:"locked_odds"`
	ExpectedProfit   float64 `json:"expected_profit"`
	ActualProfit     float64 `json:"actual_profit"`
	Status           string  `json:"status"`
	FundLockTxHash   string  `json:"fund_lock_tx_hash,omitempty"`
	SettlementTxHash string  `json:"settlement_tx_hash,omitempty"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

// GetOrderDetail 按 order_uuid 获取订单详情
func (s *OrderService) GetOrderDetail(ctx context.Context, orderUUID string) (*OrderDetail, error) {
	o, err := s.orderRepo.GetByUUID(ctx, orderUUID)
	if err != nil {
		return nil, err
	}
	detail := &OrderDetail{
		OrderUUID:      o.OrderUUID,
		UserWallet:     o.UserWallet,
		EventID:        o.EventID,
		BetOption:      o.BetOption,
		BetAmount:      o.BetAmount,
		LockedOdds:     o.LockedOdds,
		ExpectedProfit: o.ExpectedProfit,
		ActualProfit:   o.ActualProfit,
		Status:         o.Status,
		CreatedAt:      o.CreatedAt.UnixMilli(),
		UpdatedAt:      o.UpdatedAt.UnixMilli(),
	}
	if o.PlatformOrderID != nil {
		detail.PlatformOrderID = *o.PlatformOrderID
	}
	if o.FundLockTxHash != nil {
		detail.FundLockTxHash = *o.FundLockTxHash
	}
	if o.SettlementTxHash != nil {
		detail.SettlementTxHash = *o.SettlementTxHash
	}
	if e, err := s.marketRepo.GetEventByID(ctx, o.EventID); err == nil && e != nil {
		detail.EventUUID = e.EventUUID
		detail.EventTitle = e.Title
	}
	detail.PlatformID = o.PlatformID
	return detail, nil
}

// OnSettlementCompleted 链上结算完成时调用：更新订单为 settled 并写入 settlement_records
func (s *OrderService) OnSettlementCompleted(ctx context.Context, orderUUID, txHash string, settlementAmount, manageFee, gasFee float64) error {
	o, err := s.orderRepo.GetByUUID(ctx, orderUUID)
	if err != nil {
		return fmt.Errorf("订单不存在: %w", err)
	}
	if err := s.orderRepo.UpdateOrderSettlement(ctx, orderUUID, txHash); err != nil {
		return err
	}
	record := &model.SettlementRecord{
		OrderUUID:        orderUUID,
		UserWallet:       o.UserWallet,
		SettlementAmount: settlementAmount,
		ManageFee:        manageFee,
		GasFee:           gasFee,
		TxHash:           txHash,
	}
	return s.orderRepo.CreateSettlementRecord(ctx, record)
}
