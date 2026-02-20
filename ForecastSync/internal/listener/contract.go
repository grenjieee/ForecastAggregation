package listener

import (
	"context"

	"ForecastSync/internal/service"

	"github.com/sirupsen/logrus"
)

// ContractListener 订阅链上下单事件并调用 OrderService.CreateOrderFromChainEvent。
// 实际实现需使用 go-ethereum 订阅合约 BetPlaced 事件，解析 user、event 标识、方向、金额、tx_hash 后构造 ChainBetEvent 并调用 OnBetPlaced。
type ContractListener struct {
	orderService *service.OrderService
	logger       *logrus.Logger
}

// NewContractListener 创建合约事件监听器
func NewContractListener(orderService *service.OrderService, logger *logrus.Logger) *ContractListener {
	return &ContractListener{
		orderService: orderService,
		logger:       logger,
	}
}

// OnDepositSuccess 收到链上 DepositSuccess 入账事件时调用
// 仅将 contract_order_id、amount、currency 写入 contract_events，不创建 Order
// 前端调用 POST /api/orders/place 时再校验并创建订单
func (l *ContractListener) OnDepositSuccess(ctx context.Context, ev *service.DepositSuccessEvent) error {
	if ev == nil {
		return nil
	}
	err := l.orderService.SaveDepositSuccess(ctx, ev)
	if err != nil {
		l.logger.WithError(err).WithField("tx_hash", ev.TxHash).Error("SaveDepositSuccess failed")
		return err
	}
	l.logger.WithField("contract_order_id", ev.ContractOrderID).WithField("amount", ev.Amount).Info("DepositSuccess saved")
	return nil
}

// OnBetPlaced 收到链上 BetPlaced 事件时调用（由实际订阅逻辑解析后调用）
func (l *ContractListener) OnBetPlaced(ctx context.Context, ev *service.ChainBetEvent) error {
	if ev == nil {
		return nil
	}
	err := l.orderService.CreateOrderFromChainEvent(ctx, ev)
	if err != nil {
		l.logger.WithError(err).WithField("tx_hash", ev.TxHash).Error("CreateOrderFromChainEvent failed")
		return err
	}
	return nil
}

// OnSettlementCompleted 链上结算完成时调用：更新订单为 settled 并写入 settlement_records
func (l *ContractListener) OnSettlementCompleted(ctx context.Context, orderUUID, txHash string, settlementAmount, manageFee, gasFee float64) error {
	return l.orderService.OnSettlementCompleted(ctx, orderUUID, txHash, settlementAmount, manageFee, gasFee)
}

// Start 启动监听（stub：实际应在此处订阅链上事件并循环调用 OnBetPlaced / OnSettlementCompleted）
func (l *ContractListener) Start(ctx context.Context) error {
	l.logger.Info("ContractListener started (stub: no chain subscription)")
	<-ctx.Done()
	return nil
}
