package model

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement;comment:自增主键ID"`
	WalletAddress string    `gorm:"column:wallet_address;type:varchar(64);uniqueIndex;not null;comment:用户钱包地址"`
	TotalProfit   float64   `gorm:"column:total_profit;type:numeric(18,6);default:0;comment:累计盈利"`
	TotalLoss     float64   `gorm:"column:total_loss;type:numeric(18,6);default:0;comment:累计亏损"`
	TotalFee      float64   `gorm:"column:total_fee;type:numeric(18,6);default:0;comment:累计平台管理费"`
	GasFeeTotal   float64   `gorm:"column:gas_fee_total;type:numeric(18,6);default:0;comment:累计Gas费"`
	IsActive      bool      `gorm:"column:is_active;type:boolean;default:true;comment:是否活跃"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamp;default:now();comment:创建时间"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamp;default:now();comment:更新时间"`
}

type Platform struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement;comment:自增主键ID"`
	Name            string    `gorm:"column:name;type:varchar(32);not null;comment:平台名称"`
	Type            string    `gorm:"column:type;type:varchar(16);not null;comment:平台类型：chain/centralized"`
	ApiUrl          string    `gorm:"column:api_url;type:varchar(256);comment:API地址"`
	ContractAddress string    `gorm:"column:contract_address;type:varchar(64);comment:合约地址"`
	RpcUrl          string    `gorm:"column:rpc_url;type:varchar(256);comment:RPC地址"`
	ApiKey          string    `gorm:"column:api_key;type:varchar(128);comment:API密钥"`
	ApiLimit        int       `gorm:"column:api_limit;type:int;default:600;comment:API调用限额"`
	CurrentApiUsage int       `gorm:"column:current_api_usage;type:int;default:0;comment:已调用次数"`
	IsHot           bool      `gorm:"column:is_hot;type:boolean;default:false;comment:是否热门"`
	IsEnabled       bool      `gorm:"column:is_enabled;type:boolean;default:true;comment:是否启用"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamp;default:now();comment:创建时间"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamp;default:now();comment:更新时间"`
}

type Event struct {
	ID              uint64         `gorm:"column:id;primaryKey;autoIncrement;comment:自增主键ID"`
	EventUUID       string         `gorm:"column:event_uuid;type:varchar(64);uniqueIndex;not null;comment:全局唯一ID"`
	Title           string         `gorm:"column:title;type:varchar(256);not null;comment:事件标题"`
	Type            string         `gorm:"column:type;type:varchar(16);not null;comment:事件类型：sports/politics"`
	PlatformID      uint64         `gorm:"column:platform_id;type:bigint;not null;comment:关联平台ID"`
	PlatformEventID string         `gorm:"column:platform_event_id;type:varchar(64);not null;comment:平台原生ID"`
	StartTime       time.Time      `gorm:"column:start_time;type:timestamp;not null;comment:开始时间"`
	EndTime         time.Time      `gorm:"column:end_time;type:timestamp;not null;comment:结束时间"`
	ResolveTime     *time.Time     `gorm:"column:resolve_time;type:timestamp;comment:结果公布时间"`
	Options         datatypes.JSON `gorm:"column:options;type:jsonb;not null;comment:下注选项"`
	Result          *string        `gorm:"column:result;type:varchar(32);comment:最终结果"`
	ResultSource    *string        `gorm:"column:result_source;type:varchar(64);comment:结果来源"`
	ResultVerified  bool           `gorm:"column:result_verified;type:boolean;default:false;comment:结果是否核验"`
	Status          string         `gorm:"column:status;type:varchar(16);default:active;comment:状态：active/resolved/canceled"`
	IsHot           bool           `gorm:"column:is_hot;type:boolean;default:false;comment:是否热门"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:timestamp;default:now();comment:创建时间"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:timestamp;default:now();comment:更新时间"`
}

type EventOdds struct {
	ID                  uint64          `gorm:"column:id;primaryKey;autoIncrement;comment:自增主键ID"`
	EventID             uint64          `gorm:"column:event_id;type:bigint;not null;comment:关联事件ID"`
	PlatformID          uint64          `gorm:"column:platform_id;type:bigint;not null;comment:关联平台ID"`
	Odds                datatypes.JSON  `gorm:"column:odds;type:jsonb;not null;comment:各选项赔率"`
	Fee                 float64         `gorm:"column:fee;type:numeric(8,4);default:0;comment:手续费比例"`
	MaxBet              *float64        `gorm:"column:max_bet;type:numeric(18,6);comment:最大下注金额"`
	MinBet              float64         `gorm:"column:min_bet;type:numeric(18,6);default:0.1;comment:最小下注金额"`
	LockedOdds          *datatypes.JSON `gorm:"column:locked_odds;type:jsonb;comment:历史锁定赔率"`
	CacheLevel          string          `gorm:"column:cache_level;type:varchar(8);default:db;comment:缓存层级"`
	UpdatedAt           time.Time       `gorm:"column:updated_at;type:timestamp;default:now();comment:更新时间"`
	UniqueEventPlatform struct{}        `gorm:"uniqueIndex:uk_event_platform;comment:事件+平台唯一"`
}

func (User) TableName() string      { return "users" }
func (Platform) TableName() string  { return "platforms" }
func (Event) TableName() string     { return "events" }
func (EventOdds) TableName() string { return "event_odds" }
