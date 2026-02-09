package repository

import (
	"context"

	"ForecastSync/internal/model"

	"gorm.io/gorm"
)

// OrderRepository 订单持久化
type OrderRepository interface {
	CreateOrder(ctx context.Context, order *model.Order) error
}

// ContractEventRepository 合约事件持久化
type ContractEventRepository interface {
	SaveContractEvent(ctx context.Context, ev *model.ContractEvent) error
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

func (r *orderRepository) SaveContractEvent(ctx context.Context, ev *model.ContractEvent) error {
	return r.db.WithContext(ctx).Create(ev).Error
}
