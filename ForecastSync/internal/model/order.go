package model

import (
	"time"

	"gorm.io/datatypes"
)

// ContractEvent 对应 contract_events 表，用于记录链上事件原始数据
type ContractEvent struct {
	ID          uint64         `gorm:"column:id;primaryKey;autoIncrement"`
	EventType   string         `gorm:"column:event_type;type:varchar(32);not null"`
	OrderUUID   string         `gorm:"column:order_uuid;type:varchar(64);not null"`
	UserWallet  string         `gorm:"column:user_wallet;type:varchar(64);not null"`
	TxHash      string         `gorm:"column:tx_hash;type:varchar(66);uniqueIndex;not null"`
	BlockNumber *int64         `gorm:"column:block_number"`
	EventData   datatypes.JSON `gorm:"column:event_data;type:jsonb;not null"`
	Processed   bool           `gorm:"column:processed;type:boolean;default:false"`
	ProcessedAt *time.Time     `gorm:"column:processed_at"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:timestamp;default:now()"`
}

func (ContractEvent) TableName() string { return "contract_events" }

// Order 对应 orders 表，记录聚合后实际下注的订单
type Order struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderUUID        string    `gorm:"column:order_uuid;type:varchar(64);uniqueIndex;not null"`
	UserWallet       string    `gorm:"column:user_wallet;type:varchar(64);not null"`
	EventID          uint64    `gorm:"column:event_id;type:bigint;not null"`
	PlatformID       uint64    `gorm:"column:platform_id;type:bigint;not null"`
	PlatformOrderID  *string   `gorm:"column:platform_order_id;type:varchar(64)"`
	BetOption        string    `gorm:"column:bet_option;type:varchar(32);not null"`
	BetAmount        float64   `gorm:"column:bet_amount;type:numeric(18,6);not null"`
	LockedOdds       float64   `gorm:"column:locked_odds;type:numeric(10,2);not null"`
	ExpectedProfit   float64   `gorm:"column:expected_profit;type:numeric(18,6);default:0"`
	ActualProfit     float64   `gorm:"column:actual_profit;type:numeric(18,6);default:0"`
	PlatformFee      float64   `gorm:"column:platform_fee;type:numeric(18,6);default:0"`
	ManageFee        float64   `gorm:"column:manage_fee;type:numeric(18,6);default:0"`
	GasFee           float64   `gorm:"column:gas_fee;type:numeric(18,6);default:0"`
	FundLockTxHash   *string   `gorm:"column:fund_lock_tx_hash;type:varchar(66)"`
	SettlementTxHash *string   `gorm:"column:settlement_tx_hash;type:varchar(66)"`
	Status           string    `gorm:"column:status;type:varchar(16);default:'pending_lock'"`
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamp;default:now()"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:timestamp;default:now()"`
}

func (Order) TableName() string { return "orders" }
