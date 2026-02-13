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

// EventsStreamer 可选接口：按批流式拉取事件，每批通过 yield 交给调用方（如另一协程落库），避免全量进内存。
// 实现方需保证同一场赛事跨批去重（如同一 platform_event_id 只出现在一个 batch 中一次）。
type EventsStreamer interface {
	FetchEventsWithYield(ctx context.Context, eventType string, yield func(batch []*model.PlatformRawEvent) error) (total int, err error)
}

// EventResultFetcher 可选：拉取已结束事件的结果，用于结果同步与订单结算
type EventResultFetcher interface {
	FetchEventResult(ctx context.Context, platformEventID string) (result, status string, err error)
}

// PlatformRepository 通用数据库操作接口
type PlatformRepository interface {
	SaveEvents(ctx context.Context, events []*model.Event, odds []*model.EventOdds) error
}
