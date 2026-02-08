package model

import (
	"time"

	"gorm.io/gorm"
)

// PlatformType 平台类型枚举
type PlatformType string

const (
	PlatformPolymarket PlatformType = "polymarket"
	PlatformKalshi     PlatformType = "kalshi" // 新增！Kalshi官方拼写
	PlatformAugur3     PlatformType = "augur3"
)

// SportEvent 统一的体育赛事模型（抹平各平台差异）
type SportEvent struct {
	ID            string         `gorm:"primaryKey;column:id;type:varchar(64)"`           // 全局唯一ID（平台+原始ID）
	Platform      PlatformType   `gorm:"column:platform;type:varchar(32);index;not null"` // 来源平台
	OriginalID    string         `gorm:"column:original_id;type:varchar(64);not null"`    // 平台原始ID
	Name          string         `gorm:"column:name;type:varchar(255);not null"`          // 赛事名称
	SportType     string         `gorm:"column:sport_type;type:varchar(64);not null"`     // 体育类型（足球/篮球等）
	Status        string         `gorm:"column:status;type:varchar(32);not null"`         // 赛事状态
	StartTime     time.Time      `gorm:"column:start_time;type:timestamp;not null"`       // 开始时间
	Expiration    time.Time      `gorm:"column:expiration;type:datetime;not null"`        // 到期/结束时间
	RuleDesc      string         `gorm:"column:rule_desc;type:text"`                      // 规则描述
	WinCondition  string         `gorm:"column:win_condition;type:text"`                  // 赢判定条件
	LossCondition string         `gorm:"column:loss_condition;type:text"`                 // 输判定条件
	DrawCondition string         `gorm:"column:draw_condition;type:text"`                 // 平判定条件
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime"`                // 创建时间
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime"`                // 更新时间
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`                         // 软删除
}

// TableName 指定统一赛事表名
func (s *SportEvent) TableName() string {
	return "sport_events"
}

// SportEventOption 统一的赛事选项模型（输赢平）
type SportEventOption struct {
	ID          string         `gorm:"primaryKey;column:id;type:varchar(64)"`           // 全局唯一ID
	EventID     string         `gorm:"column:event_id;type:varchar(64);index;not null"` // 关联赛事ID
	Platform    PlatformType   `gorm:"column:platform;type:varchar(32);not null"`       // 来源平台
	OriginalID  string         `gorm:"column:original_id;type:varchar(64);not null"`    // 平台原始选项ID
	Name        string         `gorm:"column:name;type:varchar(64);not null"`           // 选项名称（赢/输/平）
	Price       float64        `gorm:"column:price;type:decimal(10,4);not null"`        // 价格/赔率
	Description string         `gorm:"column:description;type:text"`                    // 选项描述
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`                // 创建时间
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`                // 更新时间
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`                         // 软删除
}

// TableName 指定统一选项表名
func (s *SportEventOption) TableName() string {
	return "sport_event_options"
}
