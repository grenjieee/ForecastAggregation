package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ForecastSync/internal/chain"
	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"
	"ForecastSync/internal/repository"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	db               *gorm.DB
	logger           *logrus.Logger
	marketRepo       repository.MarketRepository
	canonicalRepo    repository.CanonicalRepository
	orderRepo        repository.OrderRepository
	contractEvents   repository.ContractEventRepository
	eventRepo        *repository.EventRepository
	tradingAdapters  map[uint64]interfaces.TradingAdapter  // platformID -> adapter，可为 nil
	liveOddsFetchers map[uint64]interfaces.LiveOddsFetcher // platformID -> 实时赔率拉取，可为 nil 则用 DB 赔率
	fiatConversion   FiatConversionService                 // Kalshi 下单前 USDC->USD，可为 nil 则用占位
	chainCfg         *config.ChainConfig                   // 解冻时调用 Escrow.releaseFunds，nil 则不可解冻
}

// NewOrderService 创建 OrderService。tradingAdapters 可为 nil，则不调用真实下单
func NewOrderService(db *gorm.DB, logger *logrus.Logger, tradingAdapters map[uint64]interfaces.TradingAdapter) *OrderService {
	return NewOrderServiceWithDeps(db, logger, tradingAdapters, nil, nil, nil, nil)
}

// NewOrderServiceWithDeps 创建 OrderService，支持注入 FiatConversion、EventRepo、LiveOddsFetchers、ChainConfig（解冻用）
func NewOrderServiceWithDeps(db *gorm.DB, logger *logrus.Logger, tradingAdapters map[uint64]interfaces.TradingAdapter, fiat FiatConversionService, eventRepo *repository.EventRepository, liveOddsFetchers map[uint64]interfaces.LiveOddsFetcher, chainCfg *config.ChainConfig) *OrderService {
	if fiat == nil {
		fiat = NewNoopFiatConversion()
	}
	return &OrderService{
		db:               db,
		logger:           logger,
		marketRepo:       repository.NewMarketRepository(db),
		canonicalRepo:    repository.NewCanonicalRepository(db),
		orderRepo:        repository.NewOrderRepository(db),
		contractEvents:   repository.NewContractEventRepository(db),
		eventRepo:        eventRepo,
		tradingAdapters:  tradingAdapters,
		liveOddsFetchers: liveOddsFetchers,
		fiatConversion:   fiat,
		chainCfg:         chainCfg,
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

// pickBestOdds 在所有赔率中挑选 BetOption（YES/NO 或平台原名）对应的最高价格，返回平台原始 option_name 供下单请求使用。
func pickBestOdds(odds []*model.EventOdds, betOption string) (platformID uint64, price float64, optionName string, err error) {
	betOption = strings.Trim(betOption, " ")
	if betOption == "" {
		return 0, 0, "", fmt.Errorf("betOption 不能为空")
	}
	betUpper := strings.ToUpper(betOption)

	var (
		found bool
		best  float64
		pid   uint64
		name  string
	)

	for _, o := range odds {
		// 匹配：选项名一致，或 YES/NO 与 option_type win/lose 对应（保留各平台原始 option_name，下单时用原名请求）
		optionUpper := strings.ToUpper(strings.Trim(o.OptionName, " "))
		nameMatch := optionUpper == betUpper
		winLoseMatch := (betUpper == "YES" && o.OptionType == "win") || (betUpper == "NO" && o.OptionType == "lose")
		if !nameMatch && !winLoseMatch {
			continue
		}
		if !found || o.Price > best {
			found = true
			best = o.Price
			pid = o.PlatformID
			name = o.OptionName // 返回平台原始名称，供 Polymarket/Kalshi 等直接用原名解析 token 或下单
		}
	}

	if !found {
		return 0, 0, "", fmt.Errorf("未找到匹配下注方向的赔率: bet_option=%s", betOption)
	}

	return pid, best, name, nil
}

// clampOddsForSign 赔率 100%→0.99、0%→0.01，用于待签名消息与返回给前端的 locked_odds，避免平台拒单
func clampOddsForSign(price float64) float64 {
	if price >= 1 {
		return 0.99
	}
	if price <= 0 {
		return 0.01
	}
	return price
}

// PlaceOrderRequest 前端下单请求
type PlaceOrderRequest struct {
	ContractOrderID string  `json:"contract_order_id"` // 合约生成的订单号
	EventUUID       string  `json:"event_uuid"`        // 本系统赛事 event_uuid 或 canonical_id
	BetOption       string  `json:"bet_option"`        // YES/NO
	Amount          float64 `json:"amount,omitempty"`  // 可选，用于与合约事件金额校验
	// 前端可传 clamp 后的锁定赔率：100% 传 0.99、0% 传 0.01，避免平台拒单；不传则用实时最佳赔率并 clamp
	LockedOdds    float64 `json:"locked_odds,omitempty"`
	MessageToSign string  `json:"message_to_sign,omitempty"`
	Signature     string  `json:"signature,omitempty"`
}

// PlaceOrderResult 下单结果
type PlaceOrderResult struct {
	OrderUUID       string `json:"order_uuid"`
	PlatformOrderID string `json:"platform_order_id"`
	PlatformID      uint64 `json:"platform_id"`
	Status          string `json:"status"`
}

// PrepareOrderRequest 获取待签名信息请求（与 Place 参数一致，用于先查赔率再签名再下单）
type PrepareOrderRequest struct {
	ContractOrderID string `json:"contract_order_id"`
	EventUUID       string `json:"event_uuid"`
	BetOption       string `json:"bet_option"`
}

// PrepareOrderResult 返回实时最佳赔率与待签名消息
type PrepareOrderResult struct {
	LockedOdds    float64 `json:"locked_odds"`     // 当前实时最高赔率
	MessageToSign string  `json:"message_to_sign"` // 用户需 personal_sign 的消息
	ExpiresAtSec  int64   `json:"expires_at_sec"`  // 过期时间戳（秒）
}

const prepareOrderExpirySec = 300 // 5 分钟

// PrepareOrderFromFrontend 前端调用：实时查三方赔率，返回最高赔率与待签名消息（签名后再调 PlaceOrder）
func (s *OrderService) PrepareOrderFromFrontend(ctx context.Context, req *PrepareOrderRequest) (*PrepareOrderResult, error) {
	if req == nil || req.ContractOrderID == "" || req.EventUUID == "" || req.BetOption == "" {
		return nil, fmt.Errorf("contract_order_id, event_uuid, bet_option 必填")
	}
	_, err := s.contractEvents.GetUnprocessedByContractOrderID(ctx, req.ContractOrderID)
	if err != nil {
		if ce, getErr := s.contractEvents.GetContractEventByContractOrderID(ctx, req.ContractOrderID); getErr == nil && ce != nil {
			if ce.Processed {
				return nil, fmt.Errorf("该合约订单已下单")
			}
			if ce.RefundedAt != nil {
				return nil, fmt.Errorf("该合约订单已解冻，无法继续下单")
			}
		}
		return nil, fmt.Errorf("未找到未处理的入账事件 contract_order_id=%s: %w", req.ContractOrderID, err)
	}
	event, eventIDs, links, err := s.resolveEventAndLinks(ctx, req.EventUUID)
	if err != nil {
		return nil, err
	}
	odds, fetchedPerLink, err := s.fetchLiveOddsForEvent(ctx, event, eventIDs, links)
	if err != nil {
		return nil, err
	}
	_, bestPrice, _, err := pickBestOdds(odds, req.BetOption)
	if err != nil {
		return nil, err
	}
	_ = fetchedPerLink // 仅 Prepare 不需要写回
	// 待签名消息与返回前端的赔率用 clamp 值，避免 0/1 导致签名后下单被平台拒单
	lockedOdds := clampOddsForSign(bestPrice)
	expiresAt := time.Now().Unix() + prepareOrderExpirySec
	msg := fmt.Sprintf("PlaceOrder:%s:%s:%s:%.6f:%d", req.ContractOrderID, req.EventUUID, req.BetOption, lockedOdds, expiresAt)
	return &PrepareOrderResult{
		LockedOdds:    lockedOdds,
		MessageToSign: msg,
		ExpiresAtSec:  expiresAt,
	}, nil
}

// resolveEventAndLinks 根据 event_uuid 解析出 event、eventIDs、links
func (s *OrderService) resolveEventAndLinks(ctx context.Context, eventUUID string) (*model.Event, []uint64, []*model.EventPlatformLink, error) {
	event, err := s.marketRepo.GetEventByUUID(ctx, eventUUID)
	if err != nil {
		if id, parseErr := strconv.ParseUint(eventUUID, 10, 64); parseErr == nil {
			links, linkErr := s.canonicalRepo.ListLinksByCanonicalID(ctx, id)
			if linkErr != nil || len(links) == 0 {
				return nil, nil, nil, fmt.Errorf("event_uuid 或 canonical_id 无效: %w", err)
			}
			event, err = s.marketRepo.GetEventByID(ctx, links[0].EventID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("查询事件失败: %w", err)
			}
		} else {
			return nil, nil, nil, fmt.Errorf("查询事件失败 event_uuid=%s: %w", eventUUID, err)
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
	return event, eventIDs, links, nil
}

// linkOdds 用于 fetchLiveOddsForEvent
type linkOdds struct {
	eventID         uint64
	platformID      uint64
	platformEventID string
	rows            []interfaces.LiveOddsRow
}

// fetchLiveOddsForEvent 拉取该赛事在多平台的实时赔率
func (s *OrderService) fetchLiveOddsForEvent(ctx context.Context, event *model.Event, eventIDs []uint64, links []*model.EventPlatformLink) ([]*model.EventOdds, []linkOdds, error) {
	var fetchedPerLink []linkOdds
	var odds []*model.EventOdds
	if s.liveOddsFetchers != nil {
		if len(links) > 0 {
			for _, l := range links {
				ev, _ := s.marketRepo.GetEventByID(ctx, l.EventID)
				if ev == nil {
					continue
				}
				fetcher := s.liveOddsFetchers[l.PlatformID]
				if fetcher == nil {
					continue
				}
				rows, err := fetcher.FetchLiveOdds(ctx, l.PlatformID, ev.PlatformEventID)
				if err != nil {
					s.logger.WithError(err).WithFields(logrus.Fields{"platform_id": l.PlatformID, "platform_event_id": ev.PlatformEventID}).Warn("拉取实时赔率失败，跳过该平台")
					continue
				}
				fetchedPerLink = append(fetchedPerLink, linkOdds{eventID: l.EventID, platformID: l.PlatformID, platformEventID: ev.PlatformEventID, rows: rows})
				for _, r := range rows {
					odds = append(odds, &model.EventOdds{PlatformID: r.PlatformID, OptionName: r.OptionName, Price: r.Price})
				}
			}
		} else {
			fetcher := s.liveOddsFetchers[event.PlatformID]
			if fetcher != nil {
				rows, err := fetcher.FetchLiveOdds(ctx, event.PlatformID, event.PlatformEventID)
				if err == nil {
					fetchedPerLink = append(fetchedPerLink, linkOdds{eventID: event.ID, platformID: event.PlatformID, platformEventID: event.PlatformEventID, rows: rows})
					for _, r := range rows {
						odds = append(odds, &model.EventOdds{PlatformID: r.PlatformID, OptionName: r.OptionName, Price: r.Price})
					}
				}
			}
		}
	}
	if len(odds) == 0 {
		var err error
		odds, err = s.marketRepo.GetOddsByEventIDs(ctx, eventIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("查询赔率失败: %w", err)
		}
	}
	if len(odds) == 0 {
		return nil, nil, fmt.Errorf("该赛事暂无可用赔率")
	}
	return odds, fetchedPerLink, nil
}

// verifyOrderSignature 校验 personal_sign(messageToSign) 的签名者是否为 userWallet
func verifyOrderSignature(userWallet, messageToSign, signatureHex string) error {
	if userWallet == "" || messageToSign == "" || signatureHex == "" {
		return fmt.Errorf("user_wallet, message_to_sign, signature 必填")
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(signatureHex, "0x"))
	if err != nil || len(sig) < 65 {
		return fmt.Errorf("invalid signature hex")
	}
	// 钱包 personal_sign 返回的 v 多为 27/28，go-ethereum SigToPub 期望 recovery id 0/1
	sigCopy := make([]byte, 65)
	copy(sigCopy, sig)
	if sigCopy[64] == 27 || sigCopy[64] == 28 {
		sigCopy[64] -= 27
	}
	hash := crypto.Keccak256Hash([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len(messageToSign)) + messageToSign))
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigCopy)
	if err != nil {
		return fmt.Errorf("signature recovery failed: %w", err)
	}
	recovered := crypto.PubkeyToAddress(*pubKey).Hex()
	if !strings.EqualFold(recovered, userWallet) {
		return fmt.Errorf("签名者与入账钱包不一致: %s vs %s", recovered, userWallet)
	}
	// 解析 message 中的过期时间 PlaceOrder:...:...:...:...:expires_at
	parts := strings.Split(messageToSign, ":")
	if len(parts) < 6 {
		return fmt.Errorf("message_to_sign 格式无效")
	}
	expiresAt, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return fmt.Errorf("message_to_sign 过期时间无效: %w", err)
	}
	if time.Now().Unix() > expiresAt {
		return fmt.Errorf("待签名消息已过期")
	}
	return nil
}

// PlaceOrderFromFrontend 前端调用：校验 contract_order_id 对应入账事件，选平台，Kalshi 时调 Circle 占位，下单并落库
func (s *OrderService) PlaceOrderFromFrontend(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResult, error) {
	if req == nil || req.ContractOrderID == "" || req.EventUUID == "" || req.BetOption == "" {
		return nil, fmt.Errorf("contract_order_id, event_uuid, bet_option 必填")
	}

	// 1. 查未处理的 DepositSuccess 入账事件（未解冻）
	ce, err := s.contractEvents.GetUnprocessedByContractOrderID(ctx, req.ContractOrderID)
	if err != nil {
		if ev, getErr := s.contractEvents.GetContractEventByContractOrderID(ctx, req.ContractOrderID); getErr == nil && ev != nil {
			if ev.Processed {
				return nil, fmt.Errorf("该合约订单已下单")
			}
			if ev.RefundedAt != nil {
				return nil, fmt.Errorf("该合约订单已解冻，无法下单")
			}
		}
		return nil, fmt.Errorf("未找到未处理的入账事件 contract_order_id=%s: %w", req.ContractOrderID, err)
	}

	// 若前端带了签名，先校验再继续（用户签名后后端才真实下单）
	if req.Signature != "" {
		if err := verifyOrderSignature(ce.UserWallet, req.MessageToSign, req.Signature); err != nil {
			return nil, fmt.Errorf("签名校验失败: %w", err)
		}
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

	// 2. 解析 event 与 links，并实时拉取赔率
	event, eventIDs, links, err := s.resolveEventAndLinks(ctx, req.EventUUID)
	if err != nil {
		return nil, err
	}
	odds, fetchedPerLink, err := s.fetchLiveOddsForEvent(ctx, event, eventIDs, links)
	if err != nil {
		return nil, err
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

	// 6. 调用 TradingAdapter 下单：优先使用前端传来的 locked_odds（前端已做 100%→0.99、0%→0.01），否则用实时最佳赔率
	lockedOdds := bestPrice
	if req.LockedOdds > 0 {
		lockedOdds = req.LockedOdds
	}
	platformOrderID := ""
	if s.tradingAdapters != nil {
		if adapter := s.tradingAdapters[bestPlatformID]; adapter != nil {
			platformOrderID, err = adapter.PlaceOrder(ctx, &interfaces.PlaceOrderRequest{
				PlatformID:      bestPlatformID,
				PlatformEventID: targetEvent.PlatformEventID,
				BetOption:       bestOptionName,
				BetAmount:       betAmountUSD,
				LockedOdds:      lockedOdds,
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

	// 9. 将本次拉取的实时赔率写回 event_odds，便于列表/详情展示最新赔率
	if s.eventRepo != nil && len(fetchedPerLink) > 0 {
		var oddsRows []repository.OddsRow
		for _, link := range fetchedPerLink {
			for _, r := range link.rows {
				oddsRows = append(oddsRows, repository.OddsRow{
					EventID:         link.eventID,
					PlatformID:      link.platformID,
					PlatformEventID: link.platformEventID,
					OptionName:      r.OptionName,
					Price:           r.Price,
				})
			}
		}
		if err := s.eventRepo.UpsertOddsForEvents(ctx, oddsRows); err != nil {
			s.logger.WithError(err).Warn("UpsertOddsForEvents failed")
		}
	}

	return &PlaceOrderResult{
		OrderUUID:       req.ContractOrderID,
		PlatformOrderID: platformOrderID,
		PlatformID:      bestPlatformID,
		Status:          "placed",
	}, nil
}

// RequestUnfreeze 申请解冻：校验存在未处理且未解冻的入账后调用 Escrow.releaseFunds，并标记已解冻。可选 wallet 用于校验入账钱包一致。
func (s *OrderService) RequestUnfreeze(ctx context.Context, contractOrderID string, wallet string) (txHash string, err error) {
	if contractOrderID == "" {
		return "", fmt.Errorf("contract_order_id 必填")
	}
	if s.chainCfg == nil || s.chainCfg.ExecutorPrivateKey == "" || s.chainCfg.EscrowAddress == "" || s.chainCfg.RPCURL == "" {
		return "", fmt.Errorf("解冻未配置链参数（rpc_url、escrow_address、CHAIN_EXECUTOR_PRIVATE_KEY）")
	}

	ce, err := s.contractEvents.GetUnprocessedByContractOrderID(ctx, contractOrderID)
	if err != nil {
		return "", fmt.Errorf("未找到可解冻的入账记录，可能已下单或已解冻")
	}
	if wallet != "" && ce.UserWallet != wallet {
		return "", fmt.Errorf("入账钱包与请求 wallet 不一致")
	}
	amount := 0.0
	if ce.DepositAmount != nil {
		amount = *ce.DepositAmount
	}
	if amount <= 0 {
		return "", fmt.Errorf("入账金额无效")
	}
	amountBig := chain.FloatToUSDCAmount(amount)
	if amountBig.Sign() <= 0 {
		return "", fmt.Errorf("入账金额无效")
	}
	toAddr := common.HexToAddress(ce.UserWallet)
	txHash, err = chain.ReleaseFunds(ctx, s.chainCfg.RPCURL, s.chainCfg.EscrowAddress, s.chainCfg.ExecutorPrivateKey, contractOrderID, toAddr, amountBig)
	if err != nil {
		return "", fmt.Errorf("链上解冻失败: %w", err)
	}
	if err := s.contractEvents.MarkRefundedByContractOrderID(ctx, contractOrderID); err != nil {
		s.logger.WithError(err).WithField("contract_order_id", contractOrderID).Warn("MarkRefundedByContractOrderID failed after tx sent")
		// 交易已发出，仍返回 txHash，仅记录告警
	}
	return txHash, nil
}

// ContractOrderStatus 返回合约订单状态：unprocessed（可下单/可解冻）、placed（已下单）、refunded（已解冻）、not_found（无入账记录）
func (s *OrderService) ContractOrderStatus(ctx context.Context, contractOrderID string) (status string, err error) {
	if contractOrderID == "" {
		return "", fmt.Errorf("contract_order_id 必填")
	}
	ce, err := s.contractEvents.GetContractEventByContractOrderID(ctx, contractOrderID)
	if err != nil {
		return "not_found", nil
	}
	if ce.RefundedAt != nil {
		return "refunded", nil
	}
	if ce.Processed {
		return "placed", nil
	}
	return "unprocessed", nil
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

// WithdrawInfo 提现所需参数；type=chain 时前端用 contract_address/method 让用户签名；type=kalshi 时后端处理
type WithdrawInfo struct {
	OrderUUID       string  `json:"order_uuid"`
	UserWallet      string  `json:"user_wallet"`
	Type            string  `json:"type"`                  // "chain" | "kalshi"
	Amount          float64 `json:"amount"`                // 总可提现（链上）或 payout（Kalshi）
	Fee             float64 `json:"fee,omitempty"`         // Kalshi 1% 手续费
	UserAmount      float64 `json:"user_amount,omitempty"` // Kalshi 用户实得
	ContractAddress string  `json:"contract_address"`      // 链上提现时合约地址
	Method          string  `json:"method"`
	Message         string  `json:"message"`
}

const kalshiPlatformID = 2
const feeRateBps = 100 // 1% = 100 bps

// GetWithdrawInfo 获取订单提现参数（仅 status=settled 可提现）；Kalshi 返回 type=kalshi 与 fee/user_amount
func (s *OrderService) GetWithdrawInfo(ctx context.Context, orderUUID string) (*WithdrawInfo, error) {
	o, err := s.orderRepo.GetByUUID(ctx, orderUUID)
	if err != nil {
		return nil, err
	}
	if o.Status != "settled" {
		return nil, fmt.Errorf("订单状态 %s 不可提现，需为 settled", o.Status)
	}
	payout := o.BetAmount + o.ActualProfit
	if payout < 0 {
		payout = 0
	}
	if o.PlatformID == kalshiPlatformID {
		profit := o.ActualProfit
		if profit < 0 {
			profit = 0
		}
		fee := profit * float64(feeRateBps) / 10000
		userAmount := payout - fee
		return &WithdrawInfo{
			OrderUUID:  o.OrderUUID,
			UserWallet: o.UserWallet,
			Type:       "kalshi",
			Amount:     payout,
			Fee:        fee,
			UserAmount: userAmount,
			Message:    "后端将处理提现（Circle USD→USDC，1% 手续费入 FeeVault）",
		}, nil
	}
	return &WithdrawInfo{
		OrderUUID:       o.OrderUUID,
		UserWallet:      o.UserWallet,
		Type:            "chain",
		Amount:          payout,
		ContractAddress: "", // 从配置读取
		Method:          "withdraw",
		Message:         "用户签名并支付 Gas 完成链上提现，Gas 费由用户承担",
	}, nil
}

// RequestWithdraw 用户发起提现：Kalshi 由后端计算并标记 withdrawn（实际打款需配置 Circle+链上）；链上由前端签名
func (s *OrderService) RequestWithdraw(ctx context.Context, orderUUID string) error {
	o, err := s.orderRepo.GetByUUID(ctx, orderUUID)
	if err != nil {
		return err
	}
	if o.Status != "settled" {
		return fmt.Errorf("订单状态 %s 不可提现，需为 settled", o.Status)
	}
	if o.PlatformID == kalshiPlatformID {
		return s.processKalshiWithdraw(ctx, o)
	}
	return s.orderRepo.UpdateOrderStatus(ctx, orderUUID, "withdraw_requested")
}

// processKalshiWithdraw 计算 1% 手续费与用户实得，更新订单为 withdrawn；实际打款需配置链上热钱包或 Circle payout
func (s *OrderService) processKalshiWithdraw(ctx context.Context, o *model.Order) error {
	payout := o.BetAmount + o.ActualProfit
	if payout < 0 {
		payout = 0
	}
	profit := o.ActualProfit
	if profit < 0 {
		profit = 0
	}
	fee := profit * float64(feeRateBps) / 10000
	_ = fee
	_ = payout
	// TODO: 调用 Circle ConvertFromUSD(payout) 得到 USDC 数量，再链上 transfer(user, userAmount), transfer(feeVault, fee)
	// 当前仅更新状态，实际打款需配置 chain.fee_vault_address 与热钱包或 Circle 打款 API
	return s.orderRepo.UpdateOrderStatus(ctx, o.OrderUUID, "withdrawn")
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
