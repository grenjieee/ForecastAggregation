package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"ForecastSync/internal/repository"

	"github.com/sirupsen/logrus"
)

// MarketService 面向前端的市场聚合服务
type MarketService struct {
	repo          repository.MarketRepository
	canonicalRepo repository.CanonicalRepository
	logger        *logrus.Logger
}

// NewMarketService 创建 MarketService
func NewMarketService(repo repository.MarketRepository, canonicalRepo repository.CanonicalRepository, logger *logrus.Logger) *MarketService {
	return &MarketService{
		repo:          repo,
		canonicalRepo: canonicalRepo,
		logger:        logger,
	}
}

// MarketSummary 列表页单个市场信息（基于聚合赛事，每场一条）
type MarketSummary struct {
	CanonicalID   int64   `json:"canonical_id"` // 聚合赛事数字 ID
	Title         string  `json:"title"`
	HomeTeam      string  `json:"home_team"`
	AwayTeam      string  `json:"away_team"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	MatchTime     int64   `json:"match_time"` // 开赛时间戳（毫秒）
	StartTime     int64   `json:"start_time"` // 兼容
	EndTime       int64   `json:"end_time"`   // 兼容
	PlatformCount int     `json:"platform_count"`
	BestPrice     float64 `json:"best_price"`
	BestPricePlat string  `json:"best_price_platform"`
	BestPriceOpt  string  `json:"best_price_option"`
	// 三档赔率（option_type 归一化后）
	WinOdds  float64 `json:"win_odds,omitempty"`
	DrawOdds float64 `json:"draw_odds,omitempty"`
	LoseOdds float64 `json:"lose_odds,omitempty"`
	// 兼容：若前端仍用 event_uuid，可填第一个平台的 event_uuid
	EventUUID string `json:"event_uuid,omitempty"`
}

// MarketListResult 列表返回
type MarketListResult struct {
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
	Items    []MarketSummary `json:"items"`
}

// ListMarkets 按条件分页返回市场列表（基于聚合赛事，每场比赛一条）
func (s *MarketService) ListMarkets(ctx context.Context, filter repository.MarketFilter, page, pageSize int) (*MarketListResult, error) {
	cf := repository.CanonicalFilter{
		SportType: filter.Type,
		Status:    filter.Status,
	}
	canonicals, total, err := s.canonicalRepo.ListCanonicalEvents(ctx, cf, page, pageSize)
	if err != nil {
		return nil, err
	}
	if len(canonicals) == 0 {
		return &MarketListResult{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Items:    []MarketSummary{},
		}, nil
	}

	platforms, err := s.repo.GetPlatforms(ctx)
	if err != nil {
		return nil, err
	}
	platNameByID := make(map[uint64]string, len(platforms))
	for _, p := range platforms {
		platNameByID[p.ID] = p.Name
	}

	result := &MarketListResult{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Items:    make([]MarketSummary, 0, len(canonicals)),
	}

	for _, ce := range canonicals {
		links, err := s.canonicalRepo.ListLinksByCanonicalID(ctx, ce.ID)
		if err != nil {
			s.logger.WithError(err).WithField("canonical_id", ce.ID).Warn("ListLinksByCanonicalID")
			continue
		}
		eventIDs := make([]uint64, 0, len(links))
		var firstEventUUID string
		for _, l := range links {
			eventIDs = append(eventIDs, l.EventID)
			if firstEventUUID == "" {
				e, _ := s.repo.GetEventByID(ctx, l.EventID)
				if e != nil {
					firstEventUUID = e.EventUUID
				}
			}
		}
		if len(eventIDs) == 0 {
			continue
		}
		odds, err := s.repo.GetOddsByEventIDs(ctx, eventIDs)
		if err != nil {
			s.logger.WithError(err).Warn("GetOddsByEventIDs")
			continue
		}
		platformSet := make(map[uint64]struct{})
		var bestPrice float64
		var bestPlatName, bestOptName string
		var winOdds, drawOdds, loseOdds float64
		for _, o := range odds {
			platformSet[o.PlatformID] = struct{}{}
			if o.Price > bestPrice {
				bestPrice = o.Price
				bestOptName = o.OptionName
				bestPlatName = platNameByID[o.PlatformID]
			}
			switch o.OptionType {
			case "win":
				if o.Price > winOdds {
					winOdds = o.Price
				}
			case "draw":
				if o.Price > drawOdds {
					drawOdds = o.Price
				}
			case "lose":
				if o.Price > loseOdds {
					loseOdds = o.Price
				}
			}
		}
		matchTime := ce.MatchTime.UnixMilli()
		summary := MarketSummary{
			CanonicalID:   int64(ce.ID),
			Title:         ce.Title,
			HomeTeam:      ce.HomeTeam,
			AwayTeam:      ce.AwayTeam,
			Type:          ce.SportType,
			Status:        ce.Status,
			MatchTime:     matchTime,
			StartTime:     matchTime,
			EndTime:       matchTime,
			PlatformCount: len(platformSet),
			BestPrice:     bestPrice,
			BestPricePlat: bestPlatName,
			BestPriceOpt:  bestOptName,
			WinOdds:       winOdds,
			DrawOdds:      drawOdds,
			LoseOdds:      loseOdds,
			EventUUID:     firstEventUUID,
		}
		result.Items = append(result.Items, summary)
	}

	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].MatchTime < result.Items[j].MatchTime
	})

	return result, nil
}

// ===== 详情页 DTO =====

type PlatformOption struct {
	PlatformID   uint64  `json:"platform_id"`
	PlatformName string  `json:"platform_name"`
	OptionName   string  `json:"option_name"`
	Price        float64 `json:"price"`
}

type MarketDetail struct {
	Event struct {
		EventUUID string `json:"event_uuid"`
		Title     string `json:"title"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		StartTime int64  `json:"start_time"`
		EndTime   int64  `json:"end_time"`
	} `json:"event"`

	Options []PlatformOption `json:"platform_options"`

	Analytics struct {
		BestPrice      float64 `json:"best_price"`
		BestPricePlat  string  `json:"best_price_platform"`
		BestPriceOpt   string  `json:"best_price_option"`
		PlatformCount  int     `json:"platform_count"`
		OptionCount    int     `json:"option_count"`
		PriceMin       float64 `json:"price_min"`
		PriceMax       float64 `json:"price_max"`
		PriceSpreadPct float64 `json:"price_spread_pct"` // (max-min)/max
	} `json:"analytics"`
}

// GetMarketDetail 获取单个市场详情。idOrEventUUID 为数字时当作 canonical_id，否则当作 event_uuid 查询所属聚合赛事。
func (s *MarketService) GetMarketDetail(ctx context.Context, idOrEventUUID string) (*MarketDetail, error) {
	var canonicalID uint64
	if idOrEventUUID == "" {
		return nil, fmt.Errorf("id or event_uuid is required")
	}
	// 尝试解析为数字 canonical_id
	if n, err := strconv.ParseUint(idOrEventUUID, 10, 64); err == nil {
		canonicalID = n
	} else {
		// 按 event_uuid 查事件，再查所属 canonical_id
		event, err := s.repo.GetEventByUUID(ctx, idOrEventUUID)
		if err != nil {
			return nil, err
		}
		canonicalID, err = s.canonicalRepo.GetCanonicalIDByEventID(ctx, event.ID)
		if err != nil {
			return nil, err
		}
	}
	return s.GetMarketDetailByCanonicalID(ctx, canonicalID)
}

// GetMarketDetailByCanonicalID 按聚合赛事 ID 返回多平台详情与赔率对比
func (s *MarketService) GetMarketDetailByCanonicalID(ctx context.Context, canonicalID uint64) (*MarketDetail, error) {
	ce, err := s.canonicalRepo.GetCanonicalByID(ctx, canonicalID)
	if err != nil {
		return nil, err
	}
	links, err := s.canonicalRepo.ListLinksByCanonicalID(ctx, canonicalID)
	if err != nil {
		return nil, err
	}
	eventIDs := make([]uint64, 0, len(links))
	for _, l := range links {
		eventIDs = append(eventIDs, l.EventID)
	}
	odds, err := s.repo.GetOddsByEventIDs(ctx, eventIDs)
	if err != nil {
		return nil, err
	}
	platforms, err := s.repo.GetPlatforms(ctx)
	if err != nil {
		return nil, err
	}
	platNameByID := make(map[uint64]string, len(platforms))
	for _, p := range platforms {
		platNameByID[p.ID] = p.Name
	}

	detail := &MarketDetail{}
	detail.Event.EventUUID = "" // 聚合详情无单一 event_uuid
	detail.Event.Title = ce.Title
	detail.Event.Type = ce.SportType
	detail.Event.Status = ce.Status
	detail.Event.StartTime = ce.MatchTime.UnixMilli()
	detail.Event.EndTime = ce.MatchTime.UnixMilli()

	platformSet := make(map[uint64]struct{})
	var bestPrice, minPrice, maxPrice float64
	var bestPlatName, bestOptName string

	for i, o := range odds {
		platformSet[o.PlatformID] = struct{}{}
		po := PlatformOption{
			PlatformID:   o.PlatformID,
			PlatformName: platNameByID[o.PlatformID],
			OptionName:   o.OptionName,
			Price:        o.Price,
		}
		detail.Options = append(detail.Options, po)

		if i == 0 {
			minPrice = o.Price
			maxPrice = o.Price
		}
		if o.Price < minPrice {
			minPrice = o.Price
		}
		if o.Price > maxPrice {
			maxPrice = o.Price
		}
		if o.Price > bestPrice {
			bestPrice = o.Price
			bestOptName = o.OptionName
			bestPlatName = platNameByID[o.PlatformID]
		}
	}

	detail.Analytics.BestPrice = bestPrice
	detail.Analytics.BestPricePlat = bestPlatName
	detail.Analytics.BestPriceOpt = bestOptName
	detail.Analytics.PlatformCount = len(platformSet)
	detail.Analytics.OptionCount = len(detail.Options)
	detail.Analytics.PriceMin = minPrice
	detail.Analytics.PriceMax = maxPrice
	if maxPrice > 0 {
		detail.Analytics.PriceSpreadPct = (maxPrice - minPrice) / maxPrice
	}

	return detail, nil
}
