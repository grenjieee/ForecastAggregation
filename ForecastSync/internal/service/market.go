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

// OutcomeItem YES/NO 等选项（用于 UI 展示百分比）
type OutcomeItem struct {
	Label string  `json:"label"` // YES / NO
	Price float64 `json:"price"` // 0-1 概率
	Pct   int     `json:"pct"`   // 0-100 百分比，便于前端直接展示
}

// MarketSummary 列表页单个市场信息（一期仅 Sports，适配 UI 卡片）
type MarketSummary struct {
	CanonicalID   int64         `json:"canonical_id"`        // 聚合赛事 ID，Compare 链接用
	Title         string        `json:"title"`               // 市场标题，如 "Lakers win NBA Championship 2026?"
	Description   string        `json:"description"`         // 详细描述，可同 title 或生成
	Type          string        `json:"type"`                // 一期固定 "sports"
	Status        string        `json:"status"`              // active / resolved
	EndTime       int64         `json:"end_time"`            // 结束时间戳（毫秒），前端格式化为 "Jul 1"
	PlatformCount int           `json:"platform_count"`      // 可用平台数，如 3
	Volume        float64       `json:"volume"`              // 交易量，前端格式化为 "$1.9M"
	SavePct       float64       `json:"save_pct"`            // 最优价比参考价节省百分比，如 20.0
	BestPricePlat string        `json:"best_price_platform"` // 最优价平台名，如 "Kalshi"
	Outcomes      []OutcomeItem `json:"outcomes"`            // YES/NO 百分比，如 [{label:"YES",pct:16},{label:"NO",pct:84}]
	EventUUID     string        `json:"event_uuid"`          // 首平台 event_uuid，Compare 链接备用
}

// MarketListResult 列表返回
type MarketListResult struct {
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
	Items    []MarketSummary `json:"items"`
}

// ListMarkets 按条件分页返回市场列表（一期仅 Sports，基于聚合赛事，适配 UI 卡片）
func (s *MarketService) ListMarkets(ctx context.Context, filter repository.MarketFilter, page, pageSize int) (*MarketListResult, error) {
	cf := repository.CanonicalFilter{
		SportType: "sports", // 一期固定 sports
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
		var totalVolume float64
		var bestPrice, minPrice, maxPrice float64
		var bestPlatID uint64
		firstPrice := true
		platOdds := make(map[uint64]map[string]float64) // platformID -> optionName -> price
		for _, o := range odds {
			platformSet[o.PlatformID] = struct{}{}
			totalVolume += o.Volume
			if firstPrice {
				minPrice, maxPrice = o.Price, o.Price
				firstPrice = false
			}
			if o.Price < minPrice {
				minPrice = o.Price
			}
			if o.Price > maxPrice {
				maxPrice = o.Price
			}
			if o.Price > bestPrice {
				bestPrice = o.Price
				bestPlatID = o.PlatformID
			}
			if platOdds[o.PlatformID] == nil {
				platOdds[o.PlatformID] = make(map[string]float64)
			}
			platOdds[o.PlatformID][o.OptionName] = o.Price
		}

		// 最优平台的 YES/NO（或首两档）作为 outcomes
		var outcomes []OutcomeItem
		if m, ok := platOdds[bestPlatID]; ok {
			if yesP, ok := m["YES"]; ok {
				pct := int(yesP * 100)
				if pct > 100 {
					pct = 100
				}
				outcomes = append(outcomes, OutcomeItem{Label: "YES", Price: yesP, Pct: pct})
			}
			if noP, ok := m["NO"]; ok {
				pct := int(noP * 100)
				if pct > 100 {
					pct = 100
				}
				outcomes = append(outcomes, OutcomeItem{Label: "NO", Price: noP, Pct: pct})
			}
			if len(outcomes) == 0 {
				for opt, p := range m {
					pct := int(p * 100)
					if pct > 100 {
						pct = 100
					}
					outcomes = append(outcomes, OutcomeItem{Label: opt, Price: p, Pct: pct})
				}
			}
		}

		// save_pct: 最优价 vs 最差价的节省比例，(max-min)/max*100
		savePct := 0.0
		if maxPrice > 0 && maxPrice > minPrice {
			savePct = (maxPrice - minPrice) / maxPrice * 100
		}

		// description: 有主客队则生成，否则用 title
		desc := ce.Title
		if ce.HomeTeam != "" && ce.AwayTeam != "" {
			desc = "Will " + ce.HomeTeam + " beat " + ce.AwayTeam + "?"
		}

		endTime := ce.MatchTime.UnixMilli()
		summary := MarketSummary{
			CanonicalID:   int64(ce.ID),
			Title:         ce.Title,
			Description:   desc,
			Type:          "sports",
			Status:        ce.Status,
			EndTime:       endTime,
			PlatformCount: len(platformSet),
			Volume:        totalVolume,
			SavePct:       savePct,
			BestPricePlat: platNameByID[bestPlatID],
			Outcomes:      outcomes,
			EventUUID:     firstEventUUID,
		}
		result.Items = append(result.Items, summary)
	}

	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].EndTime < result.Items[j].EndTime
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
		Volume         float64 `json:"volume"` // 交易量，与列表一致
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
	var bestPrice, minPrice, maxPrice, totalVolume float64
	var bestPlatName, bestOptName string

	for i, o := range odds {
		totalVolume += o.Volume
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
	detail.Analytics.Volume = totalVolume
	detail.Analytics.PriceMin = minPrice
	detail.Analytics.PriceMax = maxPrice
	if maxPrice > 0 {
		detail.Analytics.PriceSpreadPct = (maxPrice - minPrice) / maxPrice
	}

	return detail, nil
}
