package interfaces

import (
	"context"

	"ForecastSync/internal/model"
)

// PlatformAdapter 所有平台必须实现的核心接口
type PlatformAdapter interface {
	GetName() string                                                                                               // 平台名称
	FetchEvents(ctx context.Context, eventType string) ([]*model.PlatformRawEvent, error)                          // 爬取事件
	ConvertToDBModel(raw []*model.PlatformRawEvent, platformID uint64) ([]*model.Event, []*model.EventOdds, error) // 转换为数据库模型
}

// PlatformRepository 通用数据库操作接口
type PlatformRepository interface {
	SaveEvents(ctx context.Context, events []*model.Event, odds []*model.EventOdds) error
}
