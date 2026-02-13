package repository

import (
	"context"
	"time"

	"ForecastSync/internal/model"

	"gorm.io/gorm"
)

// OrderRepository 订单持久化
type OrderRepository interface {
	CreateOrder(ctx context.Context, order *model.Order) error
	UpdatePlatformOrderIDAndStatus(ctx context.Context, orderUUID, platformOrderID, status string) error
	ListByUser(ctx context.Context, userWallet string, page, pageSize int) ([]*model.Order, int64, error)
	GetByUUID(ctx context.Context, orderUUID string) (*model.Order, error)
	ListOrdersByEventID(ctx context.Context, eventID uint64) ([]*model.Order, error)
	UpdateOrderStatus(ctx context.Context, orderUUID, status string) error
	UpdateOrderSettlement(ctx context.Context, orderUUID, settlementTxHash string) error
	CreateSettlementRecord(ctx context.Context, record *model.SettlementRecord) error
}

// ContractEventRepository 合约事件持久化
type ContractEventRepository interface {
	SaveContractEvent(ctx context.Context, ev *model.ContractEvent) error
	UpdateOrderUUIDAndProcessed(ctx context.Context, txHash, orderUUID string) error
}

type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓储
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// NewContractEventRepository 创建合约事件仓储
func NewContractEventRepository(db *gorm.DB) ContractEventRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order *model.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *orderRepository) UpdatePlatformOrderIDAndStatus(ctx context.Context, orderUUID, platformOrderID, status string) error {
	return r.db.WithContext(ctx).Model(&model.Order{}).
		Where("order_uuid = ?", orderUUID).
		Updates(map[string]interface{}{
			"platform_order_id": platformOrderID,
			"status":            status,
			"updated_at":        time.Now(),
		}).Error
}

func (r *orderRepository) ListByUser(ctx context.Context, userWallet string, page, pageSize int) ([]*model.Order, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	db := r.db.WithContext(ctx).Model(&model.Order{}).Where("user_wallet = ?", userWallet)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Order
	if err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *orderRepository) GetByUUID(ctx context.Context, orderUUID string) (*model.Order, error) {
	var o model.Order
	if err := r.db.WithContext(ctx).Where("order_uuid = ?", orderUUID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *orderRepository) ListOrdersByEventID(ctx context.Context, eventID uint64) ([]*model.Order, error) {
	var list []*model.Order
	if err := r.db.WithContext(ctx).Where("event_id = ?", eventID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderUUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.Order{}).
		Where("order_uuid = ?", orderUUID).
		Updates(map[string]interface{}{"status": status, "updated_at": time.Now()}).Error
}

func (r *orderRepository) UpdateOrderSettlement(ctx context.Context, orderUUID, settlementTxHash string) error {
	return r.db.WithContext(ctx).Model(&model.Order{}).
		Where("order_uuid = ?", orderUUID).
		Updates(map[string]interface{}{
			"settlement_tx_hash": settlementTxHash,
			"status":             "settled",
			"updated_at":         time.Now(),
		}).Error
}

func (r *orderRepository) CreateSettlementRecord(ctx context.Context, record *model.SettlementRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *orderRepository) SaveContractEvent(ctx context.Context, ev *model.ContractEvent) error {
	return r.db.WithContext(ctx).Create(ev).Error
}

func (r *orderRepository) UpdateOrderUUIDAndProcessed(ctx context.Context, txHash, orderUUID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.ContractEvent{}).
		Where("tx_hash = ?", txHash).
		Updates(map[string]interface{}{
			"order_uuid":   orderUUID,
			"processed":    true,
			"processed_at": now,
		}).Error
}
