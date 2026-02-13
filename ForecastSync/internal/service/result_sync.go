package service

import (
	"context"
	"fmt"

	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/repository"

	"github.com/sirupsen/logrus"
)

// ResultSyncService 事件结果同步与订单状态更新（settlable/settled）
type ResultSyncService struct {
	marketRepo     repository.MarketRepository
	eventRepo      *repository.EventRepository
	orderRepo      repository.OrderRepository
	adapterFactory map[string]func(*config.PlatformConfig, *logrus.Logger) interfaces.PlatformAdapter
	cfg            *config.Config
	logger         *logrus.Logger
}

// NewResultSyncService 创建结果同步服务
func NewResultSyncService(
	marketRepo repository.MarketRepository,
	eventRepo *repository.EventRepository,
	orderRepo repository.OrderRepository,
	adapterFactory map[string]func(*config.PlatformConfig, *logrus.Logger) interfaces.PlatformAdapter,
	cfg *config.Config,
	logger *logrus.Logger,
) *ResultSyncService {
	return &ResultSyncService{
		marketRepo:     marketRepo,
		eventRepo:      eventRepo,
		orderRepo:      orderRepo,
		adapterFactory: adapterFactory,
		cfg:            cfg,
		logger:         logger,
	}
}

// Run 拉取已结束事件结果，更新 events.result/status，并将对应订单设为 settlable 或 settled
func (s *ResultSyncService) Run(ctx context.Context) error {
	events, err := s.marketRepo.ListEventsEndedButActive(ctx, 500)
	if err != nil {
		return fmt.Errorf("ListEventsEndedButActive: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	platforms, err := s.marketRepo.GetPlatforms(ctx)
	if err != nil {
		return err
	}
	platformNameByID := make(map[uint64]string)
	for _, p := range platforms {
		platformNameByID[p.ID] = p.Name
	}

	updated := 0
	for _, e := range events {
		platformName := platformNameByID[e.PlatformID]
		buildAdapter, ok := s.adapterFactory[platformName]
		if !ok {
			continue
		}
		platformCfg, ok := s.cfg.Platforms[platformName]
		if !ok {
			continue
		}
		adapter := buildAdapter(&platformCfg, s.logger)
		fetcher, ok := adapter.(interfaces.EventResultFetcher)
		if !ok {
			continue
		}
		result, status, err := fetcher.FetchEventResult(ctx, e.PlatformEventID)
		if err != nil {
			s.logger.WithError(err).WithField("event_id", e.ID).Warn("FetchEventResult")
			continue
		}
		if result == "" && status == "" {
			continue
		}
		if status != "" {
			if err := s.eventRepo.UpdateEventResult(ctx, e.ID, &result, &status); err != nil {
				s.logger.WithError(err).WithField("event_id", e.ID).Warn("UpdateEventResult")
				continue
			}
		} else if result != "" {
			if err := s.eventRepo.UpdateEventResult(ctx, e.ID, &result, nil); err != nil {
				s.logger.WithError(err).WithField("event_id", e.ID).Warn("UpdateEventResult")
				continue
			}
		}
		updated++

		orders, err := s.orderRepo.ListOrdersByEventID(ctx, e.ID)
		if err != nil {
			continue
		}
		for _, o := range orders {
			if o.Status != "placed" {
				continue
			}
			if o.BetOption == result {
				_ = s.orderRepo.UpdateOrderStatus(ctx, o.OrderUUID, "settlable")
			} else {
				_ = s.orderRepo.UpdateOrderStatus(ctx, o.OrderUUID, "settled")
			}
		}
	}

	if updated > 0 {
		s.logger.Infof("结果同步：更新 %d 个事件结果及对应订单状态", updated)
	}
	return nil
}
