package repository

import (
	"context"
	"fmt"
	"strings"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) interfaces.PlatformRepository {
	return &EventRepository{db: db}
}

// SaveEvents 纯数据库操作：仅负责事务+批量创建+UPSERT（无业务逻辑）
func (r *EventRepository) SaveEvents(ctx context.Context, events []*model.Event, odds []*model.EventOdds) error {
	// 开启事务
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	// 事务兜底
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// 1. 批量创建Event（仅补全UUID，无其他逻辑）
	for _, event := range events {
		if event.EventUUID == "" {
			event.EventUUID = uuid.NewString()
		}
	}
	if err := tx.CreateInBatches(events, 100).Error; err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("批量保存Event失败: %w", err)
	}

	// 2. 关联EventID到Odds（仅基础关联，无复杂逻辑）
	eventIDMap := make(map[string]uint64)
	for _, event := range events {
		eventIDMap[event.PlatformEventID] = event.ID
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

	// 3. 纯UPSERT操作（无去重，上层已处理）
	if len(odds) > 0 {
		err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "unique_event_platform"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"price":       gorm.Expr("EXCLUDED.price"),
				"option_name": gorm.Expr("EXCLUDED.option_name"),
				"updated_at":  gorm.Expr("EXCLUDED.updated_at"),
				// 仅保留数据库存在的字段，按需添加
			}),
		}).CreateInBatches(odds, 100).Error

		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("批量UPSERT EventOdds失败: %w", err)
		}
	}

	// 4. 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
