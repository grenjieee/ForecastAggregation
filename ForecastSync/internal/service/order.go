package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"
	"ForecastSync/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// DepositSuccessEvent 资金合约入账成功事件（DepositSuccess）
// 合约生成 order_id，监听器解析后落库 contract_events，仅做入账记录，不创建 Order
type DepositSuccessEvent struct {
	ContractOrderID string  // 合约生成的订单号
	UserWallet      string  // 用户钱包
	Amount          float64 // 入账金额
	Currency        string  // USDC/USDT/ETH
	TxHash          string  // 交易哈希
	BlockNumber     int64   // 区块高度（可选）
	RawData         map[string]interface{}
}

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
	canonicalRepo   repository.CanonicalRepository
	orderRepo       repository.OrderRepository
	contractEvents  repository.ContractEventRepository
	tradingAdapters map[uint64]interfaces.TradingAdapter // platformID -> adapter，可为 nil
	fiatConversion  FiatConversionService                // Kalshi 下单前 USDC->USD，可为 nil 则用占位
}

// NewOrderService 创建 OrderService。tradingAdapters 可为 nil，则不调用真实下单
func NewOrderService(db *gorm.DB, logger *logrus.Logger, tradingAdapters map[uint64]interfaces.TradingAdapter) *OrderService {
	return NewOrderServiceWithDeps(db, logger, tradingAdapters, nil)
}

// NewOrderServiceWithDeps 创建 OrderService，支持注入 FiatConversion
func NewOrderServiceWithDeps(db *gorm.DB, logger *logrus.Logger, tradingAdapters map[uint64]interfaces.TradingAdapter, fiat FiatConversionService) *OrderService {
	if fiat == nil {
		fiat = NewNoopFiatConversion()
	}
	return &OrderService{
		db:              db,
		logger:          logger,
		marketRepo:      repository.NewMarketRepository(db),
		canonicalRepo:   repository.NewCanonicalRepository(db),
		orderRepo:       repository.NewOrderRepository(db),
		contractEvents:  repository.NewContractEventRepository(db),
		tradingAdapters: tradingAdapters,
		fiatConversion:  fiat,
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

// SaveDepositSuccess 将入账成功事件写入 contract_events，不创建 Order
// 幂等：tx_hash 唯一，重复事件会报错（调用方可忽略）
func (s *OrderService) SaveDepositSuccess(ctx context.Context, ev *DepositSuccessEvent) error {
	if ev == nil {
		return fmt.Errorf("DepositSuccessEvent is nil")
	}
	rawBytes, _ := json.Marshal(ev.RawData)
	if rawBytes == nil {
		rawBytes = []byte("{}")
	}
	var blockNum *int64
	if ev.BlockNumber > 0 {
		blockNum = &ev.BlockNumber
	}
	ce := &model.ContractEvent{
		EventType:       "DepositSuccess",
		ContractOrderID: &ev.ContractOrderID,
		UserWallet:      ev.UserWallet,
		DepositAmount:   &ev.Amount,
		FundCurrency:    &ev.Currency,
		TxHash:          ev.TxHash,
		BlockNumber:     blockNum,
		EventData:       rawBytes,
		Processed:       false,
		CreatedAt:       time.Now(),
	}
	return s.contractEvents.SaveContractEvent(ctx, ce)
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
	betOption = strings.Trim(betOption, " ")
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
		if strings.ToUpper(strings.Trim(o.OptionName, " ")) != strings.ToUpper(betOption) {
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

// PlaceOrderRequest 前端下单请求
type PlaceOrderRequest struct {
	ContractOrderID string  `json:"contract_order_id"` // 合约生成的订单号
	EventUUID       string  `json:"event_uuid"`        // 本系统赛事 event_uuid 或 canonical_id
	BetOption       string  `json:"bet_option"`        // YES/NO
	Amount          float64 `json:"amount,omitempty"`  // 可选，用于与合约事件金额校验
}

// PlaceOrderResult 下单结果
type PlaceOrderResult struct {
	OrderUUID       string `json:"order_uuid"`
	PlatformOrderID string `json:"platform_order_id"`
	PlatformID      uint64 `json:"platform_id"`
	Status          string `json:"status"`
}

// PlaceOrderFromFrontend 前端调用：校验 contract_order_id 对应入账事件，选平台，Kalshi 时调 Circle 占位，下单并落库
func (s *OrderService) PlaceOrderFromFrontend(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResult, error) {
	if req == nil || req.ContractOrderID == "" || req.EventUUID == "" || req.BetOption == "" {
		return nil, fmt.Errorf("contract_order_id, event_uuid, bet_option 必填")
	}

	// 1. 查未处理的 DepositSuccess 入账事件
	ce, err := s.contractEvents.GetUnprocessedByContractOrderID(ctx, req.ContractOrderID)
	if err != nil {
		return nil, fmt.Errorf("未找到未处理的入账事件 contract_order_id=%s: %w", req.ContractOrderID, err)
	}

	amount := 0.0
	if ce.DepositAmount != nil {
		amount = *ce.DepositAmount
	}
	if req.Amount > 0 && amount > 0 {
		// 允许 0.01 误差
		if req.Amount-amount > 0.01 || amount-req.Amount > 0.01 {
			return nil, fmt.Errorf("金额校验失败：请求 %v 与入账 %v 不一致", req.Amount, amount)
		}
	}
	if amount <= 0 {
		return nil, fmt.Errorf("入账金额无效")
	}

	fundCurrency := "USDC"
	if ce.FundCurrency != nil && *ce.FundCurrency != "" {
		fundCurrency = *ce.FundCurrency
	}

	// 2. 根据 event_uuid 查 event，再查 canonical 下所有 event 的 odds
	event, err := s.marketRepo.GetEventByUUID(ctx, req.EventUUID)
	if err != nil {
		// 尝试按 canonical_id（数字）解析
		if id, parseErr := strconv.ParseUint(req.EventUUID, 10, 64); parseErr == nil {
			links, linkErr := s.canonicalRepo.ListLinksByCanonicalID(ctx, id)
			if linkErr != nil || len(links) == 0 {
				return nil, fmt.Errorf("event_uuid 或 canonical_id 无效: %w", err)
			}
			event, err = s.marketRepo.GetEventByID(ctx, links[0].EventID)
			if err != nil {
				return nil, fmt.Errorf("查询事件失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("查询事件失败 event_uuid=%s: %w", req.EventUUID, err)
		}
	}

	var eventIDs []uint64
	var links []*model.EventPlatformLink
	canonicalID, err := s.canonicalRepo.GetCanonicalIDByEventID(ctx, event.ID)
	if err == nil {
		links, _ = s.canonicalRepo.ListLinksByCanonicalID(ctx, canonicalID)
		for _, l := range links {
			eventIDs = append(eventIDs, l.EventID)
		}
	}
	if len(eventIDs) == 0 {
		eventIDs = []uint64{event.ID}
	}

	odds, err := s.marketRepo.GetOddsByEventIDs(ctx, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("查询赔率失败: %w", err)
	}
	if len(odds) == 0 {
		return nil, fmt.Errorf("该赛事暂无可用赔率")
	}

	// 3. 选赔率更高的平台
	bestPlatformID, bestPrice, bestOptionName, err := pickBestOdds(odds, req.BetOption)
	if err != nil {
		return nil, err
	}

	// 4. Kalshi 时调 Circle 占位（USDC/USDT/ETH -> USD）
	betAmountUSD := amount
	if bestPlatformID == 2 { // Kalshi platform_id 通常为 2
		betAmountUSD, err = s.fiatConversion.ConvertToUSD(ctx, amount, fundCurrency)
		if err != nil {
			return nil, fmt.Errorf("兑换 USD 失败: %w", err)
		}
	}

	// 5. 确定目标平台的 platform_event_id（选中的平台对应的 event）
	targetEvent := event
	for _, l := range links {
		if l.PlatformID == bestPlatformID {
			e, _ := s.marketRepo.GetEventByID(ctx, l.EventID)
			if e != nil {
				targetEvent = e
				break
			}
		}
	}
	// links 可能为空（单平台事件），使用原 event
	if len(eventIDs) > 0 {
		for _, eid := range eventIDs {
			e, _ := s.marketRepo.GetEventByID(ctx, eid)
			if e != nil && e.PlatformID == bestPlatformID {
				targetEvent = e
				break
			}
		}
	}

	// 6. 调用 TradingAdapter 下单（测试环境）
	platformOrderID := ""
	if s.tradingAdapters != nil {
		if adapter := s.tradingAdapters[bestPlatformID]; adapter != nil {
			platformOrderID, err = adapter.PlaceOrder(ctx, &interfaces.PlaceOrderRequest{
				PlatformID:      bestPlatformID,
				PlatformEventID: targetEvent.PlatformEventID,
				BetOption:       bestOptionName,
				BetAmount:       betAmountUSD,
				LockedOdds:      bestPrice,
			})
			if err != nil {
				s.logger.WithError(err).WithField("platform_id", bestPlatformID).Error("PlaceOrder failed")
				return nil, fmt.Errorf("平台下单失败: %w", err)
			}
		}
	}

	// 7. 创建 Order，order_uuid = contract_order_id
	expectedProfit := amount * (bestPrice - 1) // 简化
	if expectedProfit < 0 {
		expectedProfit = amount * (1/bestPrice - 1)
	}
	order := &model.Order{
		OrderUUID:      req.ContractOrderID,
		UserWallet:     ce.UserWallet,
		EventID:        event.ID,
		PlatformID:     bestPlatformID,
		BetOption:      bestOptionName,
		BetAmount:      amount,
		FundCurrency:   fundCurrency,
		LockedOdds:     bestPrice,
		ExpectedProfit: expectedProfit,
		Status:         "placed",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if platformOrderID != "" {
		order.PlatformOrderID = &platformOrderID
	}

	if err := s.orderRepo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}

	// 8. 标记 contract_event 已处理
	if err := s.contractEvents.UpdateProcessedByContractOrderID(ctx, req.ContractOrderID, req.ContractOrderID); err != nil {
		s.logger.WithError(err).Warn("UpdateProcessedByContractOrderID failed")
	}

	return &PlaceOrderResult{
		OrderUUID:       req.ContractOrderID,
		PlatformOrderID: platformOrderID,
		PlatformID:      bestPlatformID,
		Status:          "placed",
	}, nil
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

// ListByUser 按用户钱包分页查询订单列表。status 可选，如 status=settled 查可提现订单
func (s *OrderService) ListByUser(ctx context.Context, userWallet string, page, pageSize int) (*OrderListResult, error) {
	return s.ListByUserWithStatus(ctx, userWallet, "", page, pageSize)
}

func (s *OrderService) ListByUserWithStatus(ctx context.Context, userWallet, status string, page, pageSize int) (*OrderListResult, error) {
	orders, total, err := s.orderRepo.ListByUserWithStatus(ctx, userWallet, status, page, pageSize)
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
	OrderUUID        string  `json:"order_uuid"`        // 合约订单号
	PlatformOrderID  string  `json:"platform_order_id"` // 三方平台订单号
	UserWallet       string  `json:"user_wallet"`
	EventID          uint64  `json:"event_id"`
	EventUUID        string  `json:"event_uuid"`
	EventTitle       string  `json:"event_title"`
	PlatformID       uint64  `json:"platform_id"`
	BetOption        string  `json:"bet_option"`
	BetAmount        float64 `json:"bet_amount"`
	FundCurrency     string  `json:"fund_currency"` // USDC/USDT/ETH
	LockedOdds       float64 `json:"locked_odds"`
	ExpectedProfit   float64 `json:"expected_profit"`
	ActualProfit     float64 `json:"actual_profit"`
	Status           string  `json:"status"`
	FundLockTxHash   string  `json:"fund_lock_tx_hash,omitempty"`
	SettlementTxHash string  `json:"settlement_tx_hash,omitempty"`
	StartTime        int64   `json:"start_time"` // 盘口开始时间（毫秒）
	EndTime          int64   `json:"end_time"`   // 盘口结束时间（毫秒）
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

// GetOrderDetail 按 order_uuid 获取订单详情（含盘口时间、fund_currency）
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
		FundCurrency:   o.FundCurrency,
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
		detail.StartTime = e.StartTime.UnixMilli()
		detail.EndTime = e.EndTime.UnixMilli()
	}
	detail.PlatformID = o.PlatformID
	return detail, nil
}

// WithdrawInfo 提现所需链上参数（供前端/钱包让用户签名，Gas 由用户承担）
type WithdrawInfo struct {
	OrderUUID       string  `json:"order_uuid"`
	UserWallet      string  `json:"user_wallet"`
	Amount          float64 `json:"amount"`
	ContractAddress string  `json:"contract_address"` // 合约地址，占位
	Method          string  `json:"method"`           // 合约方法，如 withdraw
	Message         string  `json:"message"`          // 说明：用户签名并支付 Gas 完成提现
}

// GetWithdrawInfo 获取订单提现参数（仅 status=settled 可提现）
func (s *OrderService) GetWithdrawInfo(ctx context.Context, orderUUID string) (*WithdrawInfo, error) {
	o, err := s.orderRepo.GetByUUID(ctx, orderUUID)
	if err != nil {
		return nil, err
	}
	if o.Status != "settled" {
		return nil, fmt.Errorf("订单状态 %s 不可提现，需为 settled", o.Status)
	}
	// 计算可提现金额：实际收益 + 本金 - 已扣费用，此处简化为 bet_amount + actual_profit
	amount := o.BetAmount + o.ActualProfit
	if amount < 0 {
		amount = 0
	}
	return &WithdrawInfo{
		OrderUUID:       o.OrderUUID,
		UserWallet:      o.UserWallet,
		Amount:          amount,
		ContractAddress: "", // 占位：从配置读取合约地址
		Method:          "withdraw",
		Message:         "用户签名并支付 Gas 完成链上提现，Gas 费由用户承担",
	}, nil
}

// RequestWithdraw 用户发起提现请求，更新订单状态为 withdraw_requested
// 链上执行由前端/用户完成，后端仅记录状态
func (s *OrderService) RequestWithdraw(ctx context.Context, orderUUID string) error {
	o, err := s.orderRepo.GetByUUID(ctx, orderUUID)
	if err != nil {
		return err
	}
	if o.Status != "settled" {
		return fmt.Errorf("订单状态 %s 不可提现，需为 settled", o.Status)
	}
	return s.orderRepo.UpdateOrderStatus(ctx, orderUUID, "withdraw_requested")
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
