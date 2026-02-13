package service

import (
	"ForecastSync/internal/config"
	"context"
	"fmt"
	"sync"
	"time"

	"ForecastSync/internal/adapter/kalshi"
	"ForecastSync/internal/adapter/polymarket"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"
	"ForecastSync/internal/repository"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SyncService struct {
	db             *gorm.DB
	logger         *logrus.Logger
	repo           interfaces.PlatformRepository
	cfg            *config.Config
	aggregation    *AggregationService
	resultSync     *ResultSyncService
	adapterFactory map[string]func(platformCfg *config.PlatformConfig, logger *logrus.Logger) interfaces.PlatformAdapter
}

func NewSyncService(db *gorm.DB, logger *logrus.Logger, cfg *config.Config) *SyncService {
	marketRepo := repository.NewMarketRepository(db)
	canonicalRepo := repository.NewCanonicalRepository(db)
	eventRepoInst := repository.NewEventRepositoryInstance(db)
	orderRepo := repository.NewOrderRepository(db)
	adapterFactory := map[string]func(platformCfg *config.PlatformConfig, logger *logrus.Logger) interfaces.PlatformAdapter{
		"polymarket": polymarket.NewPolymarketAdapter,
		"kalshi":     kalshi.NewKalshiAdapter,
	}
	return &SyncService{
		db:             db,
		logger:         logger,
		repo:           eventRepoInst,
		cfg:            cfg,
		aggregation:    NewAggregationService(marketRepo, canonicalRepo, logger),
		resultSync:     NewResultSyncService(marketRepo, eventRepoInst, orderRepo, adapterFactory, cfg, logger),
		adapterFactory: adapterFactory,
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

	// 4. 爬取事件：支持流式的平台用「生产者 yield + 独立协程落库」，避免全量进内存导致频繁 GC；同一场赛事各平台在适配层已做跨批去重
	var totalEvents int
	var err error
	if streamer, ok := adapter.(interfaces.EventsStreamer); ok {
		totalEvents, err = s.syncPlatformStreaming(ctx, platformName, eventType, &platform, adapter, streamer)
		if err != nil {
			return err
		}
		if totalEvents == 0 {
			s.logger.Warnf("%s未爬取到%s类型事件", platformName, eventType)
			return nil
		}
	} else {
		rawEvents, err := adapter.FetchEvents(ctx, eventType)
		if err != nil {
			return fmt.Errorf("%s爬取事件失败: %w", platformName, err)
		}
		if len(rawEvents) == 0 {
			s.logger.Warnf("%s未爬取到%s类型事件", platformName, eventType)
			return nil
		}
		events, odds, err := adapter.ConvertToDBModel(rawEvents, platform.ID)
		if err != nil {
			return fmt.Errorf("%s转换数据失败: %w", platformName, err)
		}
		uniqueOdds := s.dedupEventOdds(odds)
		if err := s.repo.SaveEvents(ctx, events, uniqueOdds); err != nil {
			return fmt.Errorf("%s入库失败: %w", platformName, err)
		}
		totalEvents = len(events)
	}

	// 7. 同步完成后执行聚合任务（更新 canonical_events + event_platform_links）
	if s.aggregation != nil {
		if err := s.aggregation.Run(ctx, eventType); err != nil {
			s.logger.WithError(err).Warn("聚合任务执行失败")
		}
	}

	// 8. 结果同步：已结束事件拉取 result，更新订单状态 settlable/settled
	if s.resultSync != nil {
		if err := s.resultSync.Run(ctx); err != nil {
			s.logger.WithError(err).Warn("结果同步执行失败")
		}
	}

	s.logger.Infof("%s同步完成，共%d个事件", platformName, totalEvents)
	return nil
}

// syncPlatformStreaming 使用流式接口：生产者协程按批 yield，独立协程消费并落库，保持同一场赛事去重（由各适配器在 yield 前完成）。
func (s *SyncService) syncPlatformStreaming(ctx context.Context, platformName string, eventType string, platform *model.Platform, adapter interfaces.PlatformAdapter, streamer interfaces.EventsStreamer) (totalEvents int, err error) {
	ch := make(chan []*model.PlatformRawEvent, 1)
	var wg sync.WaitGroup
	var saveErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		for batch := range ch {
			events, odds, convErr := adapter.ConvertToDBModel(batch, platform.ID)
			if convErr != nil {
				saveErr = fmt.Errorf("%s转换数据失败: %w", platformName, convErr)
				return
			}
			uniqueOdds := s.dedupEventOdds(odds)
			if persistErr := s.repo.SaveEvents(ctx, events, uniqueOdds); persistErr != nil {
				saveErr = fmt.Errorf("%s入库失败: %w", platformName, persistErr)
				return
			}
			totalEvents += len(events)
		}
	}()

	_, fetchErr := streamer.FetchEventsWithYield(ctx, eventType, func(batch []*model.PlatformRawEvent) error {
		ch <- batch
		return nil
	})
	close(ch)
	wg.Wait()

	if saveErr != nil {
		return totalEvents, saveErr
	}
	if fetchErr != nil {
		return totalEvents, fmt.Errorf("%s爬取事件失败: %w", platformName, fetchErr)
	}
	// 使用实际落库条数（totalEvents）与适配器返回的 total 应一致，以 totalEvents 为准
	return totalEvents, nil
}

func (s *SyncService) dedupEventOdds(odds []*model.EventOdds) []*model.EventOdds {
	if len(odds) == 0 {
		return []*model.EventOdds{}
	}

	// 用map去重，key=unique_event_platform，value=最新的Odds
	oddsMap := make(map[string]*model.EventOdds)
	for _, odd := range odds {
		// 空值兜底（防止unique_event_platform为空导致panic）
		if odd.UniqueEventPlatform == "" {
			odd.UniqueEventPlatform = fmt.Sprintf("%d_%d_%s_%d", odd.EventID, odd.PlatformID, odd.OptionName, time.Now().UnixNano())
		}

		// 保留更新时间最新的一条
		if existing, ok := oddsMap[odd.UniqueEventPlatform]; !ok || odd.UpdatedAt.After(existing.UpdatedAt) {
			oddsMap[odd.UniqueEventPlatform] = odd
		}
	}

	// 转换map为切片
	uniqueOdds := make([]*model.EventOdds, 0, len(oddsMap))
	for _, odd := range oddsMap {
		uniqueOdds = append(uniqueOdds, odd)
	}

	return uniqueOdds
}
