package model

import (
	"time"

	"gorm.io/datatypes"
)

// ContractEvent 对应 contract_events 表，用于记录链上事件原始数据。
// DepositSuccess 入账事件：合约生成 contract_order_id，监听器落库；前端调用 place 后创建 Order 并标记 processed。
// OrderUUID 可空：BetPlaced 先入库，创建订单后再回写 order_uuid 与 processed。
type ContractEvent struct {
	ID              uint64         `gorm:"column:id;primaryKey;autoIncrement"`
	EventType       string         `gorm:"column:event_type;type:varchar(32);not null"`
	ContractOrderID *string        `gorm:"column:contract_order_id;type:varchar(64);uniqueIndex"` // 合约生成的订单号（DepositSuccess）
	OrderUUID       *string        `gorm:"column:order_uuid;type:varchar(64)"`                    // 可空，place 创建订单后回写
	UserWallet      string         `gorm:"column:user_wallet;type:varchar(64);not null"`
	DepositAmount   *float64       `gorm:"column:deposit_amount;type:numeric(18,6)"` // 入账金额（DepositSuccess）
	FundCurrency    *string        `gorm:"column:fund_currency;type:varchar(16)"`    // 入账币种 USDC/USDT/ETH
	TxHash          string         `gorm:"column:tx_hash;type:varchar(66);uniqueIndex;not null"`
	BlockNumber     *int64         `gorm:"column:block_number"`
	EventData       datatypes.JSON `gorm:"column:event_data;type:jsonb;not null"`
	Processed       bool           `gorm:"column:processed;type:boolean;default:false"`
	ProcessedAt     *time.Time     `gorm:"column:processed_at"`
	RefundedAt      *time.Time     `gorm:"column:refunded_at"` // 解冻时间，非空表示该合约订单已解冻，不可再下单
	CreatedAt       time.Time      `gorm:"column:created_at;type:timestamp;default:now()"`
}

func (ContractEvent) TableName() string { return "contract_events" }

// Order 对应 orders 表，记录聚合后实际下注的订单
// OrderUUID 存储合约生成的订单号（contract_order_id），与 contract_events 关联
type Order struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderUUID        string    `gorm:"column:order_uuid;type:varchar(64);uniqueIndex;not null"` // 合约订单号，与 contract_order_id 一致
	UserWallet       string    `gorm:"column:user_wallet;type:varchar(64);not null"`
	EventID          uint64    `gorm:"column:event_id;type:bigint;not null"`
	PlatformID       uint64    `gorm:"column:platform_id;type:bigint;not null"`
	PlatformOrderID  *string   `gorm:"column:platform_order_id;type:varchar(64)"`
	BetOption        string    `gorm:"column:bet_option;type:varchar(32);not null"`
	BetAmount        float64   `gorm:"column:bet_amount;type:numeric(18,6);not null"`
	FundCurrency     string    `gorm:"column:fund_currency;type:varchar(16);default:'USDC'"` // 用户支付币种 USDC/USDT/ETH
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

// SettlementRecord 结算记录表
type SettlementRecord struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderUUID        string    `gorm:"column:order_uuid;type:varchar(64);not null"`
	UserWallet       string    `gorm:"column:user_wallet;type:varchar(64);not null"`
	SettlementAmount float64   `gorm:"column:settlement_amount;type:numeric(18,6);not null"`
	ManageFee        float64   `gorm:"column:manage_fee;type:numeric(18,6);default:0"`
	GasFee           float64   `gorm:"column:gas_fee;type:numeric(18,6);default:0"`
	TxHash           string    `gorm:"column:tx_hash;type:varchar(66);uniqueIndex;not null"`
	SettlementTime   time.Time `gorm:"column:settlement_time;type:timestamp;default:now()"`
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamp;default:now()"`
}

func (SettlementRecord) TableName() string { return "settlement_records" }
