package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) interfaces.PlatformRepository {
	return &EventRepository{db: db}
}

// NewEventRepositoryInstance 返回具体类型，供 ResultSync 等需要 UpdateEventResult 的调用方使用
func NewEventRepositoryInstance(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

// SaveEvents 事务内：按 (platform_id, platform_event_id) upsert 事件（确定性 event_uuid），再 upsert 赔率。
func (r *EventRepository) SaveEvents(ctx context.Context, events []*model.Event, odds []*model.EventOdds) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// 1. 确定性 event_uuid：规则 platform_id_platform_event_id
	for _, e := range events {
		if e.EventUUID == "" {
			e.EventUUID = fmt.Sprintf("%d_%s", e.PlatformID, e.PlatformEventID)
		}
	}

	// 2. Upsert events ON CONFLICT (platform_id, platform_event_id)
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "platform_id"}, {Name: "platform_event_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "start_time", "end_time", "status", "updated_at", "event_uuid", "options", "result", "result_source", "result_verified"}),
	}).CreateInBatches(events, 100).Error; err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("upsert events 失败: %w", err)
	}

	// 3. 冲突行不会回填 ID，按 (platform_id, platform_event_id) 查回 ID
	for _, e := range events {
		if e.ID != 0 {
			continue
		}
		if err := tx.Model(&model.Event{}).Where("platform_id = ? AND platform_event_id = ?", e.PlatformID, e.PlatformEventID).Select("id").First(e).Error; err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("查回 event id 失败: %w", err)
		}
	}

	// 4. 关联 EventID 到 Odds
	eventIDMap := make(map[string]uint64)
	for _, e := range events {
		eventIDMap[e.PlatformEventID] = e.ID
	}
	for _, odd := range odds {
		if odd.EventID == 0 {
			for platformEventID, eventID := range eventIDMap {
				if strings.Contains(odd.UniqueEventPlatform, platformEventID) {
					odd.EventID = eventID
					break
				}
			}
		}
	}

	// 5. Upsert event_odds
	if len(odds) > 0 {
		err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "unique_event_platform"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"price":       gorm.Expr("EXCLUDED.price"),
				"option_name": gorm.Expr("EXCLUDED.option_name"),
				"option_type": gorm.Expr("EXCLUDED.option_type"),
				"updated_at":  gorm.Expr("EXCLUDED.updated_at"),
			}),
		}).CreateInBatches(odds, 100).Error
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("批量 upsert event_odds 失败: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// UpdateEventResult 更新事件结果与状态（结果同步后调用）
func (r *EventRepository) UpdateEventResult(ctx context.Context, eventID uint64, result, status *string) error {
	updates := map[string]interface{}{"updated_at": time.Now()}
	if result != nil {
		updates["result"] = *result
	}
	if status != nil {
		updates["status"] = *status
	}
	return r.db.WithContext(ctx).Model(&model.Event{}).Where("id = ?", eventID).Updates(updates).Error
}
