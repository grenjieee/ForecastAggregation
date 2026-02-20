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

	// 批量拉取所有参与聚合的事件的赔率，用于从平台选项（如 Polymarket outcomes）中取比赛双方，避免从 title 误解析
	var allEventIDs []uint64
	for _, group := range groupByKey {
		for _, e := range group {
			allEventIDs = append(allEventIDs, e.ID)
		}
	}
	allOdds, err := s.marketRepo.GetOddsByEventIDs(ctx, allEventIDs)
	if err != nil {
		return fmt.Errorf("拉取事件赔率失败: %w", err)
	}
	oddsByEventID := make(map[uint64][]*model.EventOdds)
	for _, o := range allOdds {
		oddsByEventID[o.EventID] = append(oddsByEventID[o.EventID], o)
	}

	for key, group := range groupByKey {
		if len(group) == 0 {
			continue
		}
		first := group[0]
		homeTeam, awayTeam := extractTeamsFromOdds(oddsByEventID, group)
		ce := &model.CanonicalEvent{
			SportType:    eventType,
			Title:        first.Title,
			HomeTeam:     homeTeam,
			AwayTeam:     awayTeam,
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

// maxTeamLen 与 canonical_events.home_team/away_team 的 varchar(128) 一致
const maxTeamLen = 128

// extractTeamsFromOdds 从平台赔率选项中取比赛双方名称（仅当平台直接提供双方选项时有效）。
// Polymarket 的 outcomes 会落库为 event_odds.option_name（如 "Viktoriya Tomova" / "Suzan Lamens"），此处直接使用；
// Kalshi 仅有 YES/NO，无法得到队名，返回空。对无双方结构的比赛保持为空。
func extractTeamsFromOdds(oddsByEventID map[uint64][]*model.EventOdds, group []*model.Event) (homeTeam, awayTeam string) {
	trunc := func(s string) string {
		s = strings.TrimSpace(s)
		if len(s) > maxTeamLen {
			return s[:maxTeamLen]
		}
		return s
	}
	for _, e := range group {
		odds := oddsByEventID[e.ID]
		if len(odds) != 2 {
			continue
		}
		a, b := odds[0].OptionName, odds[1].OptionName
		// 排除 Kalshi 的 YES/NO，只使用平台提供的真实双方名称（如 Polymarket outcomes）
		if (strings.EqualFold(a, "YES") && strings.EqualFold(b, "NO")) ||
			(strings.EqualFold(a, "NO") && strings.EqualFold(b, "YES")) {
			continue
		}
		// 顺序按 option_name 字典序固定，保证主客稳定
		if strings.Compare(a, b) > 0 {
			a, b = b, a
		}
		return trunc(a), trunc(b)
	}
	return "", ""
}
