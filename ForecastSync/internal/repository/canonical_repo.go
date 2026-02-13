package repository

import (
	"context"
	"time"

	"ForecastSync/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CanonicalRepository 聚合赛事仓储
type CanonicalRepository interface {
	UpsertCanonicalEvent(ctx context.Context, ce *model.CanonicalEvent) error
	EnsureLink(ctx context.Context, canonicalEventID, eventID, platformID uint64) error
	ListLinksByCanonicalID(ctx context.Context, canonicalID uint64) ([]*model.EventPlatformLink, error)
	ListCanonicalEvents(ctx context.Context, filter CanonicalFilter, page, pageSize int) ([]*model.CanonicalEvent, int64, error)
	GetCanonicalByID(ctx context.Context, id uint64) (*model.CanonicalEvent, error)
	// GetCanonicalIDByEventID 通过 event_id 查所属聚合赛事 id（用于 by-event/:event_uuid 兼容）
	GetCanonicalIDByEventID(ctx context.Context, eventID uint64) (uint64, error)
}

// CanonicalFilter 聚合赛事列表筛选
type CanonicalFilter struct {
	SportType string     // 运动类型
	Status    string     // 状态
	FromTime  *time.Time // 开赛时间起
	ToTime    *time.Time // 开赛时间止
}

type canonicalRepository struct {
	db *gorm.DB
}

func NewCanonicalRepository(db *gorm.DB) CanonicalRepository {
	return &canonicalRepository{db: db}
}

func (r *canonicalRepository) UpsertCanonicalEvent(ctx context.Context, ce *model.CanonicalEvent) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "canonical_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "home_team", "away_team", "match_time", "status", "updated_at"}),
	}).Create(ce).Error; err != nil {
		return err
	}
	if ce.ID == 0 {
		if err := r.db.WithContext(ctx).Model(ce).Where("canonical_key = ?", ce.CanonicalKey).Select("id").First(ce).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *canonicalRepository) EnsureLink(ctx context.Context, canonicalEventID, eventID, platformID uint64) error {
	link := &model.EventPlatformLink{
		CanonicalEventID: canonicalEventID,
		EventID:          eventID,
		PlatformID:       platformID,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "canonical_event_id"}, {Name: "platform_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"event_id"}),
	}).Create(link).Error
}

func (r *canonicalRepository) ListLinksByCanonicalID(ctx context.Context, canonicalID uint64) ([]*model.EventPlatformLink, error) {
	var links []*model.EventPlatformLink
	if err := r.db.WithContext(ctx).Where("canonical_event_id = ?", canonicalID).Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

func (r *canonicalRepository) ListCanonicalEvents(ctx context.Context, filter CanonicalFilter, page, pageSize int) ([]*model.CanonicalEvent, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	db := r.db.WithContext(ctx).Model(&model.CanonicalEvent{})
	if filter.SportType != "" {
		db = db.Where("sport_type = ?", filter.SportType)
	}
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if filter.FromTime != nil {
		db = db.Where("match_time >= ?", *filter.FromTime)
	}
	if filter.ToTime != nil {
		db = db.Where("match_time <= ?", *filter.ToTime)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.CanonicalEvent
	if err := db.Order("match_time ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *canonicalRepository) GetCanonicalByID(ctx context.Context, id uint64) (*model.CanonicalEvent, error) {
	var ce model.CanonicalEvent
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&ce).Error; err != nil {
		return nil, err
	}
	return &ce, nil
}

func (r *canonicalRepository) GetCanonicalIDByEventID(ctx context.Context, eventID uint64) (uint64, error) {
	var link model.EventPlatformLink
	if err := r.db.WithContext(ctx).Where("event_id = ?", eventID).First(&link).Error; err != nil {
		return 0, err
	}
	return link.CanonicalEventID, nil
}
