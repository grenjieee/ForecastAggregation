package repository

import (
	"context"
	"time"

	"ForecastSync/internal/model"

	"gorm.io/gorm"
)

// MarketFilter 列表筛选条件
type MarketFilter struct {
	Type     string // 事件类型：sports / politics ...
	Status   string // 事件状态：active / resolved / ...
	Platform string // 可选：主平台名称（暂按 events.platform_id 对应的平台）
}

// MarketRepository 面向前端聚合查询的仓储接口
type MarketRepository interface {
	// ListEvents 按过滤条件分页查询事件
	ListEvents(ctx context.Context, filter MarketFilter, page, pageSize int) ([]*model.Event, int64, error)
	// ListEventsForAggregation 按类型拉取事件（供聚合任务用，带 limit）
	ListEventsForAggregation(ctx context.Context, eventType string, limit int) ([]*model.Event, error)
	// ListEventsEndedButActive 已过结束时间仍为 active 的事件（供结果同步）
	ListEventsEndedButActive(ctx context.Context, limit int) ([]*model.Event, error)
	// GetEventByUUID 通过 event_uuid 获取事件
	GetEventByUUID(ctx context.Context, eventUUID string) (*model.Event, error)
	// GetOddsByEventIDs 批量查询事件对应的赔率
	GetOddsByEventIDs(ctx context.Context, eventIDs []uint64) ([]*model.EventOdds, error)
	// GetOddsByEventID 查询单个事件的所有赔率
	GetOddsByEventID(ctx context.Context, eventID uint64) ([]*model.EventOdds, error)
	// GetPlatforms 获取所有平台基础信息
	GetPlatforms(ctx context.Context) ([]*model.Platform, error)
	// GetEventByID 通过 event id 获取事件
	GetEventByID(ctx context.Context, eventID uint64) (*model.Event, error)
}

type marketRepository struct {
	db *gorm.DB
}

// EventOddsView 只暴露给 service 的轻量视图结构，避免在 service 中依赖 gorm 标签
type EventOddsView struct {
	EventID    uint64
	PlatformID uint64
	OptionName string
	Price      float64
}

// NewMarketRepository 创建 MarketRepository 实例
func NewMarketRepository(db *gorm.DB) MarketRepository {
	return &marketRepository{db: db}
}

// ListEvents 按过滤条件分页查询事件
func (r *marketRepository) ListEvents(ctx context.Context, filter MarketFilter, page, pageSize int) ([]*model.Event, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	db := r.db.WithContext(ctx).Model(&model.Event{})

	if filter.Type != "" {
		db = db.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}

	// 目前简单按 events.platform_id 过滤主平台
	if filter.Platform != "" {
		// 通过平台名称过滤
		db = db.Joins("JOIN platforms ON platforms.id = events.platform_id").
			Where("platforms.name = ?", filter.Platform)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var events []*model.Event
	if err := db.
		Order("end_time ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// ListEventsForAggregation 按类型拉取事件，用于聚合任务
func (r *marketRepository) ListEventsForAggregation(ctx context.Context, eventType string, limit int) ([]*model.Event, error) {
	if limit <= 0 {
		limit = 2000
	}
	var events []*model.Event
	db := r.db.WithContext(ctx).Model(&model.Event{})
	if eventType != "" {
		db = db.Where("type = ?", eventType)
	}
	if err := db.Order("start_time ASC").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// ListEventsEndedButActive 已过 end_time 且 status=active 的事件
func (r *marketRepository) ListEventsEndedButActive(ctx context.Context, limit int) ([]*model.Event, error) {
	if limit <= 0 {
		limit = 500
	}
	var events []*model.Event
	if err := r.db.WithContext(ctx).Model(&model.Event{}).
		Where("status = ? AND end_time < ?", "active", time.Now()).
		Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// GetEventByUUID 通过 event_uuid 获取事件
func (r *marketRepository) GetEventByUUID(ctx context.Context, eventUUID string) (*model.Event, error) {
	var event model.Event
	if err := r.db.WithContext(ctx).
		Where("event_uuid = ?", eventUUID).
		First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// GetOddsByEventIDs 批量查询事件对应的赔率
func (r *marketRepository) GetOddsByEventIDs(ctx context.Context, eventIDs []uint64) ([]*model.EventOdds, error) {
	if len(eventIDs) == 0 {
		return []*model.EventOdds{}, nil
	}
	var odds []*model.EventOdds
	if err := r.db.WithContext(ctx).
		Where("event_id IN ?", eventIDs).
		Find(&odds).Error; err != nil {
		return nil, err
	}
	return odds, nil
}

// GetOddsByEventID 查询单个事件的所有赔率
func (r *marketRepository) GetOddsByEventID(ctx context.Context, eventID uint64) ([]*model.EventOdds, error) {
	var odds []*model.EventOdds
	if err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Find(&odds).Error; err != nil {
		return nil, err
	}
	return odds, nil
}

// GetPlatforms 获取所有平台基础信息
func (r *marketRepository) GetPlatforms(ctx context.Context) ([]*model.Platform, error) {
	var platforms []*model.Platform
	if err := r.db.WithContext(ctx).
		Find(&platforms).Error; err != nil {
		return nil, err
	}
	return platforms, nil
}

// GetEventByID 通过 event id 获取事件
func (r *marketRepository) GetEventByID(ctx context.Context, eventID uint64) (*model.Event, error) {
	var e model.Event
	if err := r.db.WithContext(ctx).Where("id = ?", eventID).First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}
