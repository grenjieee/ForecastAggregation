package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ForecastSync/internal/model"
	"ForecastSync/internal/repository"

	"github.com/sirupsen/logrus"
)

// AggregationService 聚合赛事服务：将多平台同场事件归并为 canonical_events + event_platform_links
type AggregationService struct {
	marketRepo    repository.MarketRepository
	canonicalRepo repository.CanonicalRepository
	logger        *logrus.Logger
}

func NewAggregationService(marketRepo repository.MarketRepository, canonicalRepo repository.CanonicalRepository, logger *logrus.Logger) *AggregationService {
	return &AggregationService{
		marketRepo:    marketRepo,
		canonicalRepo: canonicalRepo,
		logger:        logger,
	}
}

// Run 在同步完成后调用：按 type 拉取 events，按规范化键分组，upsert canonical_events 与 event_platform_links
func (s *AggregationService) Run(ctx context.Context, eventType string) error {
	if eventType == "" {
		eventType = "sports"
	}
	events, err := s.marketRepo.ListEventsForAggregation(ctx, eventType, 5000)
	if err != nil {
		return fmt.Errorf("拉取事件失败: %w", err)
	}
	if len(events) == 0 {
		s.logger.Info("聚合任务：无事件可聚合")
		return nil
	}

	// 按 canonical_key 分组
	groupByKey := make(map[string][]*model.Event)
	for _, e := range events {
		key := buildCanonicalKey(e.Title, e.StartTime)
		groupByKey[key] = append(groupByKey[key], e)
	}

	for key, group := range groupByKey {
		if len(group) == 0 {
			continue
		}
		first := group[0]
		ce := &model.CanonicalEvent{
			SportType:    eventType,
			Title:        first.Title,
			MatchTime:    first.StartTime,
			CanonicalKey: key,
			Status:       first.Status,
		}
		if err := s.canonicalRepo.UpsertCanonicalEvent(ctx, ce); err != nil {
			s.logger.WithError(err).WithField("canonical_key", key).Warn("upsert canonical_event 失败")
			continue
		}
		for _, e := range group {
			if err := s.canonicalRepo.EnsureLink(ctx, ce.ID, e.ID, e.PlatformID); err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"canonical_id": ce.ID,
					"event_id":     e.ID,
					"platform_id":  e.PlatformID,
				}).Warn("ensure event_platform_link 失败")
			}
		}
	}

	s.logger.Infof("聚合任务完成：%d 个事件归并为 %d 个聚合赛事", len(events), len(groupByKey))
	return nil
}

// buildCanonicalKey 规范化标题 + 开赛时间窗口（30 分钟）生成唯一键
func buildCanonicalKey(title string, startTime time.Time) string {
	normalized := normalizeTitle(title)
	slot := startTime.Truncate(30 * time.Minute).Unix()
	data := fmt.Sprintf("%s|%d", normalized, slot)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])[:32]
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9\s]+`)

func normalizeTitle(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonAlphaNum.ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
