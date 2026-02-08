package service

import (
	"ForecastSync/internal/config"
	"context"
	"fmt"

	"ForecastSync/internal/adapter/kalshi"
	"ForecastSync/internal/adapter/polymarket"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"
	"ForecastSync/internal/repository"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SyncService struct {
	db     *gorm.DB
	logger *logrus.Logger
	repo   interfaces.PlatformRepository
	cfg    *config.Config
	// 适配器工厂：新增平台仅需添加此处
	adapterFactory map[string]func(platformCfg *config.PlatformConfig, logger *logrus.Logger) interfaces.PlatformAdapter
}

func NewSyncService(db *gorm.DB, logger *logrus.Logger, cfg *config.Config) *SyncService {
	return &SyncService{
		db:     db,
		logger: logger,
		repo:   repository.NewEventRepository(db),
		cfg:    cfg,
		adapterFactory: map[string]func(platformCfg *config.PlatformConfig, logger *logrus.Logger) interfaces.PlatformAdapter{
			"polymarket": polymarket.NewPolymarketAdapter, // 适配小写平台名
			"kalshi":     kalshi.NewKalshiAdapter,
		},
	}
}

// SyncPlatform 通用同步方法（支持所有平台）
func (s *SyncService) SyncPlatform(ctx context.Context, platformName string, eventType string) error {
	// 1. 查询平台配置
	var platform model.Platform
	if err := s.db.WithContext(ctx).Where("name = ?", platformName).First(&platform).Error; err != nil {
		return fmt.Errorf("查询%s配置失败: %w", platformName, err)
	}
	if !platform.IsEnabled {
		return fmt.Errorf("%s平台已禁用", platformName)
	}

	// 2. 创建适配器
	adapterBuilder, ok := s.adapterFactory[platformName]
	if !ok {
		return fmt.Errorf("未支持的平台: %s", platformName)
	}
	// 3. 获取适配器对应的配置
	adapterCfg, ok := s.cfg.Platforms[platformName]
	if !ok {
		return fmt.Errorf("未获取到平台配置: %s", platformName)
	}
	adapter := adapterBuilder(&adapterCfg, s.logger)

	// 4. 爬取事件
	rawEvents, err := adapter.FetchEvents(ctx, eventType)
	if err != nil {
		return fmt.Errorf("%s爬取事件失败: %w", platformName, err)
	}
	if len(rawEvents) == 0 {
		s.logger.Warnf("%s未爬取到%s类型事件", platformName, eventType)
		return nil
	}

	// 5. 转换为数据库模型
	events, odds, err := adapter.ConvertToDBModel(rawEvents, platform.ID)
	if err != nil {
		return fmt.Errorf("%s转换数据失败: %w", platformName, err)
	}

	// 6. 通用入库
	if err := s.repo.SaveEvents(ctx, events, odds); err != nil {
		return fmt.Errorf("%s入库失败: %w", platformName, err)
	}

	s.logger.Infof("%s同步完成，共%d个事件", platformName, len(events))
	return nil
}
