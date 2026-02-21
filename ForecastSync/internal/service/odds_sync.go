package service

import (
	"context"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/repository"

	"github.com/sirupsen/logrus"
)

// OddsSyncService 定时从各平台拉取当前赔率并 upsert 到 event_odds
type OddsSyncService struct {
	marketRepo       repository.MarketRepository
	eventRepo        *repository.EventRepository
	liveOddsFetchers map[uint64]interfaces.LiveOddsFetcher
	logger           *logrus.Logger
}

// NewOddsSyncService 创建赔率同步服务
func NewOddsSyncService(marketRepo repository.MarketRepository, eventRepo *repository.EventRepository, liveOddsFetchers map[uint64]interfaces.LiveOddsFetcher, logger *logrus.Logger) *OddsSyncService {
	return &OddsSyncService{
		marketRepo:       marketRepo,
		eventRepo:        eventRepo,
		liveOddsFetchers: liveOddsFetchers,
		logger:           logger,
	}
}

// Run 拉取所有仍在交易中的事件的实时赔率并写回 event_odds；单事件失败不阻塞整次运行
func (s *OddsSyncService) Run(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 500
	}
	events, err := s.marketRepo.ListEventsActiveOpen(ctx, limit)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		s.logger.Debug("OddsSync: 无进行中事件")
		return nil
	}

	var allRows []repository.OddsRow
	for _, ev := range events {
		fetcher := s.liveOddsFetchers[ev.PlatformID]
		if fetcher == nil {
			continue
		}
		rows, err := fetcher.FetchLiveOdds(ctx, ev.PlatformID, ev.PlatformEventID)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"event_id":          ev.ID,
				"platform_id":       ev.PlatformID,
				"platform_event_id": ev.PlatformEventID,
			}).Warn("OddsSync: 拉取赔率失败，跳过")
			continue
		}
		for _, r := range rows {
			allRows = append(allRows, repository.OddsRow{
				EventID:         ev.ID,
				PlatformID:      ev.PlatformID,
				PlatformEventID: ev.PlatformEventID,
				OptionName:      r.OptionName,
				Price:           r.Price,
			})
		}
	}

	if len(allRows) == 0 {
		s.logger.Debug("OddsSync: 未拉取到任何赔率")
		return nil
	}
	if err := s.eventRepo.UpsertOddsForEvents(ctx, allRows); err != nil {
		return err
	}
	s.logger.Infof("OddsSync: 已更新 %d 条赔率", len(allRows))
	return nil
}
