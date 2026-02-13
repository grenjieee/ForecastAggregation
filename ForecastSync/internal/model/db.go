package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
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
	EventUUID       string         `gorm:"column:event_uuid;type:varchar(128);uniqueIndex;not null;comment:全局唯一ID，规则：platform_id_platform_event_id"`
	Title           string         `gorm:"column:title;type:varchar(256);not null;comment:事件标题"`
	Type            string         `gorm:"column:type;type:varchar(16);not null;comment:事件类型：sports/politics"`
	PlatformID      uint64         `gorm:"column:platform_id;type:bigint;not null;uniqueIndex:uq_platform_event;comment:关联平台ID"`
	PlatformEventID string         `gorm:"column:platform_event_id;type:varchar(128);not null;uniqueIndex:uq_platform_event;comment:平台原生ID"`
	CanonicalKey    *string        `gorm:"column:canonical_key;type:varchar(64);index;comment:聚合键，用于同场多平台归并"`
	StartTime       time.Time      `gorm:"column:start_time;type:timestamp;not null;comment:开始时间"`
	EndTime         time.Time      `gorm:"column:end_time;type:timestamp;not null;comment:结束时间"`
	ResolveTime     *time.Time     `gorm:"column:resolve_time;type:timestamp;comment:结果公布时间"`
	Options         datatypes.JSON `gorm:"column:options;type:jsonb;not null;comment:下注选项"`
	Result          *string        `gorm:"column:result;type:varchar(32);comment:最终结果"`
	ResultSource    *string        `gorm:"column:result_source;type:varchar(256);comment:结果来源"`
	ResultVerified  bool           `gorm:"column:result_verified;type:boolean;default:false;comment:结果是否核验"`
	Status          string         `gorm:"column:status;type:varchar(16);default:active;comment:状态：active/resolved/canceled"`
	IsHot           bool           `gorm:"column:is_hot;type:boolean;default:false;comment:是否热门"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:timestamp;default:now();comment:创建时间"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:timestamp;default:now();comment:更新时间"`
}

type EventOdds struct {
	ID                  uint64         `gorm:"column:id;primaryKey;autoIncrement;comment:自增主键ID"`
	EventID             uint64         `gorm:"column:event_id;type:bigint;not null;index;comment:关联事件ID"`
	UniqueEventPlatform string         `gorm:"column:unique_event_platform;type:varchar(128);uniqueIndex;not null;comment:事件+平台唯一标识"`
	PlatformID          uint64         `gorm:"column:platform_id;type:bigint;not null;comment:平台ID"`
	OptionName          string         `gorm:"column:option_name;type:varchar(64);not null;comment:赔率选项名称"`
	OptionType          string         `gorm:"column:option_type;type:varchar(16);comment:归一化选项：win/draw/lose"`
	Price               float64        `gorm:"column:price;type:decimal(10,2);not null;comment:赔率价格"` // 正确字段：price（不是odds）
	Liquidity           float64        `gorm:"column:liquidity;type:decimal(10,2);default:0;comment:流动性"`
	Volume              float64        `gorm:"column:volume;type:decimal(10,2);default:0;comment:交易量"`
	CreatedAt           time.Time      `gorm:"column:created_at;type:timestamp;default:now();comment:创建时间"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;type:timestamp;default:now();comment:更新时间"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index;comment:软删除"`
}

func (User) TableName() string      { return "users" }
func (Platform) TableName() string  { return "platforms" }
func (Event) TableName() string     { return "events" }
func (EventOdds) TableName() string { return "event_odds" }
