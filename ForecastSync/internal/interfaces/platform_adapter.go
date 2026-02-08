package interfaces

import (
	"ForecastSync/internal/config"
	"context"

	"ForecastSync/internal/model"

	"github.com/sirupsen/logrus"
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

// Factory 适配器工厂接口：根据平台名称创建对应的适配器实例
type Factory interface {
	// CreateAdapter 根据平台名称和配置创建适配器
	CreateAdapter(platformName string, platformCfg *config.PlatformConfig, logger *logrus.Logger) (PlatformAdapter, error)
}
