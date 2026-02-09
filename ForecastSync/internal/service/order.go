package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	db             *gorm.DB
	logger         *logrus.Logger
	marketRepo     repository.MarketRepository
	orderRepo      repository.OrderRepository
	contractEvents repository.ContractEventRepository
}

// NewOrderService 创建 OrderService
func NewOrderService(db *gorm.DB, logger *logrus.Logger) *OrderService {
	return &OrderService{
		db:             db,
		logger:         logger,
		marketRepo:     repository.NewMarketRepository(db),
		orderRepo:      repository.NewOrderRepository(db),
		contractEvents: repository.NewContractEventRepository(db),
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

	// 5. 生成本地订单（不直接下第三方平台单，留给后续 TradingAdapter 接入）
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
		Status:     "placed", // 资金已锁定，且已选好目标平台
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.orderRepo.CreateOrder(ctx, order); err != nil {
		return fmt.Errorf("创建订单失败: %w", err)
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
		OrderUUID:   "", // 此时本地订单尚未生成，可留空或后续通过关联补回
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
