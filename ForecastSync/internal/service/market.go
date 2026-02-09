package service

import (
	"context"
	"sort"

	"ForecastSync/internal/repository"

	"github.com/sirupsen/logrus"
)

// MarketService 面向前端的市场聚合服务
type MarketService struct {
	repo   repository.MarketRepository
	logger *logrus.Logger
}

// NewMarketService 创建 MarketService
func NewMarketService(repo repository.MarketRepository, logger *logrus.Logger) *MarketService {
	return &MarketService{
		repo:   repo,
		logger: logger,
	}
}

// MarketSummary 列表页单个市场信息（给前端用）
type MarketSummary struct {
	EventUUID     string  `json:"event_uuid"`
	Title         string  `json:"title"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	StartTime     int64   `json:"start_time"`     // 时间戳（毫秒）
	EndTime       int64   `json:"end_time"`       // 时间戳（毫秒）
	PlatformCount int     `json:"platform_count"` // 参与平台数量
	BestPrice     float64 `json:"best_price"`     // 所有选项中的最高价格
	BestPricePlat string  `json:"best_price_platform"`
	BestPriceOpt  string  `json:"best_price_option"`
}

// MarketListResult 列表返回
type MarketListResult struct {
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
	Items    []MarketSummary `json:"items"`
}

// ListMarkets 按条件分页返回市场列表
func (s *MarketService) ListMarkets(ctx context.Context, filter repository.MarketFilter, page, pageSize int) (*MarketListResult, error) {
	events, total, err := s.repo.ListEvents(ctx, filter, page, pageSize)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return &MarketListResult{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Items:    []MarketSummary{},
		}, nil
	}

	// 批量查询赔率
	eventIDs := make([]uint64, 0, len(events))
	for _, e := range events {
		eventIDs = append(eventIDs, e.ID)
	}
	odds, err := s.repo.GetOddsByEventIDs(ctx, eventIDs)
	if err != nil {
		return nil, err
	}

	// 按事件分组赔率
	oddsByEvent := make(map[uint64][]*repository.EventOddsView)
	for _, o := range odds {
		ev := &repository.EventOddsView{
			EventID:    o.EventID,
			PlatformID: o.PlatformID,
			OptionName: o.OptionName,
			Price:      o.Price,
		}
		oddsByEvent[o.EventID] = append(oddsByEvent[o.EventID], ev)
	}

	// 平台名称映射
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
		Items:    make([]MarketSummary, 0, len(events)),
	}

	for _, e := range events {
		evOdds := oddsByEvent[e.ID]
		platformSet := make(map[uint64]struct{})
		var bestPrice float64
		var bestPlatName, bestOptName string

		for _, o := range evOdds {
			platformSet[o.PlatformID] = struct{}{}
			if o.Price > bestPrice {
				bestPrice = o.Price
				bestOptName = o.OptionName
				bestPlatName = platNameByID[o.PlatformID]
			}
		}

		summary := MarketSummary{
			EventUUID:     e.EventUUID,
			Title:         e.Title,
			Type:          e.Type,
			Status:        e.Status,
			StartTime:     e.StartTime.UnixMilli(),
			EndTime:       e.EndTime.UnixMilli(),
			PlatformCount: len(platformSet),
			BestPrice:     bestPrice,
			BestPricePlat: bestPlatName,
			BestPriceOpt:  bestOptName,
		}
		result.Items = append(result.Items, summary)
	}

	// 默认按结束时间升序，如果你需要可在这里再做排序
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
		PriceMin       float64 `json:"price_min"`
		PriceMax       float64 `json:"price_max"`
		PriceSpreadPct float64 `json:"price_spread_pct"` // (max-min)/max
	} `json:"analytics"`
}

// GetMarketDetail 获取单个市场详情 + 各平台赔率对比
func (s *MarketService) GetMarketDetail(ctx context.Context, eventUUID string) (*MarketDetail, error) {
	event, err := s.repo.GetEventByUUID(ctx, eventUUID)
	if err != nil {
		return nil, err
	}

	odds, err := s.repo.GetOddsByEventID(ctx, event.ID)
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
	detail.Event.EventUUID = event.EventUUID
	detail.Event.Title = event.Title
	detail.Event.Type = event.Type
	detail.Event.Status = event.Status
	detail.Event.StartTime = event.StartTime.UnixMilli()
	detail.Event.EndTime = event.EndTime.UnixMilli()

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
