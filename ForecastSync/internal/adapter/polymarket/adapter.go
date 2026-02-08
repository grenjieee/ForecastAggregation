package polymarket

import (
	"ForecastSync/internal/config"
	"ForecastSync/internal/utils/httpclient"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"

	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

type Adapter struct {
	cfg        *config.PlatformConfig
	httpClient *http.Client
	logger     *logrus.Logger
}

func NewPolymarketAdapter(cfg *config.PlatformConfig, logger *logrus.Logger) interfaces.PlatformAdapter {
	return &Adapter{
		cfg:        cfg,
		httpClient: httpclient.NewHTTPClient(cfg, logger),
		logger:     logger,
	}
}

// GetName ========== 实现PlatformAdapter接口 ==========
func (p *Adapter) GetName() string {
	return "Polymarket"
}

func (p *Adapter) FetchEvents(ctx context.Context, eventType string) ([]*model.PlatformRawEvent, error) {
	_ = ctx
	// 1. 调用Polymarket API获取运动列表
	sportsURL := fmt.Sprintf("%s/sports", p.cfg.BaseURL)
	sportsResp, err := p.httpClient.Get(sportsURL)
	if err != nil {
		return nil, fmt.Errorf("获取运动列表失败: %w", err)
	}
	defer sportsResp.Body.Close() // 循环外的defer安全

	var sports []struct {
		Series string `json:"series"`
	}
	if err := json.NewDecoder(sportsResp.Body).Decode(&sports); err != nil {
		return nil, fmt.Errorf("解析运动列表失败: %w", err)
	}

	// 2. 筛选球类运动（eventType=sports）
	var ballSeries []string
	for _, s := range sports {
		seriesLower := strings.ToLower(s.Series)
		if strings.Contains(seriesLower, "ball") || strings.Contains(seriesLower, "cricket") {
			ballSeries = append(ballSeries, s.Series)
		}
	}

	// 3. 爬取每个系列的事件
	var rawEvents []*model.PlatformRawEvent
	for _, series := range ballSeries {
		eventsURL := fmt.Sprintf("%s/events?series_id=%s", p.cfg.BaseURL, series)
		eventsResp, err := p.httpClient.Get(eventsURL)
		if err != nil {
			p.logger.Warnf("爬取%s事件失败: %v", series, err)
			continue
		}

		// 修复：将defer放在独立代码块中，每次循环立即关闭Body（避免资源泄漏）
		// 方案1：使用匿名函数包裹，执行完立即关闭
		func() {
			defer eventsResp.Body.Close() // 此defer在匿名函数结束时执行，即当前循环迭代

			var polyEvents []model.PolymarketEvent
			if err := json.NewDecoder(eventsResp.Body).Decode(&polyEvents); err != nil {
				p.logger.Warnf("解析%s事件失败: %v", series, err)
				return // 仅退出匿名函数，不影响外层循环
			}

			// 4. 封装为通用RawEvent
			for _, e := range polyEvents {
				rawEvents = append(rawEvents, &model.PlatformRawEvent{
					Platform: p.GetName(),
					ID:       e.ID,
					Type:     eventType,
					Data:     e,
				})
			}
		}() // 立即执行匿名函数
	}

	return rawEvents, nil
}

func (p *Adapter) ConvertToDBModel(raw []*model.PlatformRawEvent, platformID uint64) ([]*model.Event, []*model.EventOdds, error) {
	var events []*model.Event
	var odds []*model.EventOdds

	for _, r := range raw {
		polyEvent, ok := r.Data.(model.PolymarketEvent)
		if !ok {
			p.logger.Warn("RawEvent数据类型错误，跳过")
			continue
		}

		// 1. 转换为Event模型
		event := &model.Event{
			Title:           polyEvent.Title,
			Type:            r.Type,
			PlatformID:      platformID,
			PlatformEventID: polyEvent.ID,
			StartTime:       polyEvent.StartDate,
			EndTime:         polyEvent.EndDate,
			Options:         p.buildOptions(polyEvent),
			Status:          p.mapStatus(polyEvent.Active, polyEvent.Closed),
			ResultSource:    &polyEvent.ResolutionSource,
		}
		events = append(events, event)

		// 2. 转换为EventOdds模型
		odd := &model.EventOdds{
			PlatformID: platformID,
			Odds:       p.buildOdds(polyEvent),
			MinBet:     p.cfg.MinBet,
			CacheLevel: "db",
			UpdatedAt:  time.Now(),
		}
		odds = append(odds, odd)
	}

	return events, odds, nil
}

func (p *Adapter) buildOptions(event model.PolymarketEvent) datatypes.JSON {
	options := make(map[string]interface{})
	for _, market := range event.Markets {
		outcomes, err := parseJSONArrayString(market.Outcomes)
		if err != nil {
			p.logger.Warnf("解析Outcomes失败: %v", err)
			continue
		}
		for _, o := range outcomes {
			options[o] = "available"
		}
	}

	// 核心修复：先序列化 map 为 JSON 字节数组
	jsonBytes, err := json.Marshal(options)
	if err != nil {
		p.logger.Warnf("序列化 options 失败: %v", err)
		return datatypes.JSON("{}") // 兜底返回空 JSON
	}
	return jsonBytes
}

func (p *Adapter) buildOdds(event model.PolymarketEvent) datatypes.JSON {
	odds := make(map[string]interface{})
	for _, market := range event.Markets {
		outcomes, err := parseJSONArrayString(market.Outcomes)
		if err != nil {
			p.logger.Warnf("解析Outcomes失败: %v", err)
			continue
		}
		prices, err := parseJSONArrayString(market.OutcomePrices)
		if err != nil {
			p.logger.Warnf("解析Prices失败: %v", err)
			continue
		}

		for i, o := range outcomes {
			if i >= len(prices) {
				continue
			}
			price, err := strconv.ParseFloat(prices[i], 64)
			if err != nil {
				p.logger.Warnf("转换赔率失败: %v", err)
				continue
			}
			odds[o] = price
		}
	}

	// 核心修复：序列化 map 为 JSON 字节数组
	jsonBytes, err := json.Marshal(odds)
	if err != nil {
		p.logger.Warnf("序列化 odds 失败: %v", err)
		return datatypes.JSON("{}") // 兜底返回空 JSON
	}
	return jsonBytes
}

func (p *Adapter) mapStatus(active, closed bool) string {
	switch {
	case active && !closed:
		return "active"
	case !active && closed:
		return "resolved"
	default:
		return "canceled"
	}
}

// 解析伪JSON数组字符串
func parseJSONArrayString(s string) ([]string, error) {
	if s == "" || s == "null" {
		return []string{}, nil
	}
	var res []string
	if err := json.Unmarshal([]byte(s), &res); err != nil {
		return nil, err
	}
	return res, nil
}
