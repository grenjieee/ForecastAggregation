package repository

import (
	"context"
	"fmt"
	"strings"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) interfaces.PlatformRepository {
	return &EventRepository{db: db}
}

// SaveEvents 通用入库逻辑（所有平台共用）
func (r *EventRepository) SaveEvents(ctx context.Context, events []*model.Event, odds []*model.EventOdds) error {
	// 开启事务
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	// 1. 保存Event
	for i := range events {
		if events[i].EventUUID == "" {
			events[i].EventUUID = uuid.NewString() // 生成全局唯一ID
		}
		if err := tx.Create(events[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("保存Event失败: %w, title: %s", err, events[i].Title)
		}
		// 关联EventID到Odds
		for j := range odds {
			if odds[j].EventID == 0 {
				odds[j].EventID = events[i].ID
			}
		}
	}

	// 2. 保存EventOdds（处理唯一约束）
	for i := range odds {
		if err := tx.Create(odds[i]).Error; err != nil {
			if strings.Contains(err.Error(), "uk_event_platform") {
				// 冲突则更新赔率
				if err := tx.Model(&model.EventOdds{}).
					Where("event_id = ? AND platform_id = ?", odds[i].EventID, odds[i].PlatformID).
					Updates(map[string]interface{}{
						"odds":       odds[i].Odds,
						"updated_at": odds[i].UpdatedAt,
					}).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("更新Odds失败: %w, event_id: %d", err, odds[i].EventID)
				}
			} else {
				tx.Rollback()
				return fmt.Errorf("保存Odds失败: %w, event_id: %d", err, odds[i].EventID)
			}
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}
