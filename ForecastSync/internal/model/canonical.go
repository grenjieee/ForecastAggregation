package model

import (
	"time"
)

// CanonicalEvent 聚合赛事主表（同一场比赛多平台去重后一条）
// id 即业务上的 canonical_id（数字，自增主键）
type CanonicalEvent struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	SportType    string    `gorm:"column:sport_type;type:varchar(64);not null"`
	Title        string    `gorm:"column:title;type:varchar(256);not null"`
	HomeTeam     string    `gorm:"column:home_team;type:varchar(128)"`
	AwayTeam     string    `gorm:"column:away_team;type:varchar(128)"`
	MatchTime    time.Time `gorm:"column:match_time;type:timestamp;not null"`
	CanonicalKey string    `gorm:"column:canonical_key;type:varchar(64);uniqueIndex;not null"` // 规范化键，用于同场判定
	Status       string    `gorm:"column:status;type:varchar(16);default:active"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamp;default:now()"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamp;default:now()"`
}

func (CanonicalEvent) TableName() string { return "canonical_events" }

// EventPlatformLink 聚合赛事与平台事件的映射
type EventPlatformLink struct {
	ID               uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	CanonicalEventID uint64 `gorm:"column:canonical_event_id;type:bigint;not null;uniqueIndex:uq_canonical_platform"`
	EventID          uint64 `gorm:"column:event_id;type:bigint;not null"`
	PlatformID       uint64 `gorm:"column:platform_id;type:bigint;not null;uniqueIndex:uq_canonical_platform"`
}

func (EventPlatformLink) TableName() string { return "event_platform_links" }
