package kalshi

import (
	"ForecastSync/internal/config"
	"ForecastSync/internal/utils/httpclient"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"

	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

const sportsSeriesCacheTTL = 4 * time.Hour

type Adapter struct {
	cfg        *config.PlatformConfig
	httpClient *http.Client
	logger     *logrus.Logger

	// 体育类 series_ticker 缓存（几小时刷新一次）
	sportsTickers   []string
	sportsTickersAt time.Time
	sportsTickersMu sync.RWMutex
}

func NewKalshiAdapter(cfg *config.PlatformConfig, logger *logrus.Logger) interfaces.PlatformAdapter {
	return &Adapter{
		cfg:        cfg,
		httpClient: httpclient.NewHTTPClient(cfg, logger),
		logger:     logger,
	}
}

// GetName ========== 实现PlatformAdapter接口 ==========
func (k *Adapter) GetName() string {
	return "Kalshi"
}

// FetchEventResult 拉取已结束事件结果（stub：可后续接 Kalshi API）
func (k *Adapter) FetchEventResult(ctx context.Context, platformEventID string) (result, status string, err error) {
	_ = ctx
	_ = platformEventID
	return "", "", nil
}

func (k *Adapter) FetchEvents(ctx context.Context, eventType string) ([]*model.PlatformRawEvent, error) {
	_ = ctx
	if eventType == "sports" {
		return k.fetchSportsEvents()
	}
	return k.fetchEventsByURL(fmt.Sprintf("%s/events?with_nested_markets=true&status=open&limit=200", k.cfg.BaseURL), eventType)
}

// FetchEventsWithYield 实现 EventsStreamer：按批流式拉取，同一 event_ticker 跨批去重（体育按 ticker 去重，非体育单批）。
func (k *Adapter) FetchEventsWithYield(ctx context.Context, eventType string, yield func(batch []*model.PlatformRawEvent) error) (total int, err error) {
	if eventType == "sports" {
		return k.FetchSportsEventsWithYield(ctx, yield)
	}
	raw, err := k.fetchEventsByURL(fmt.Sprintf("%s/events?with_nested_markets=true&status=open&limit=200", strings.TrimSuffix(k.cfg.BaseURL, "/")), eventType)
	if err != nil {
		return 0, err
	}
	if len(raw) > 0 && yield != nil {
		if err := yield(raw); err != nil {
			return 0, err
		}
		return len(raw), nil
	}
	return len(raw), nil
}

// getSportsSeriesTickers 返回体育类 series_ticker 列表（优先配置，否则走 GET /series 并缓存）
func (k *Adapter) getSportsSeriesTickers() ([]string, error) {
	if t := strings.TrimSpace(k.cfg.SeriesTicker); t != "" {
		return []string{t}, nil
	}
	k.sportsTickersMu.RLock()
	if len(k.sportsTickers) > 0 && time.Since(k.sportsTickersAt) < sportsSeriesCacheTTL {
		out := make([]string, len(k.sportsTickers))
		copy(out, k.sportsTickers)
		k.sportsTickersMu.RUnlock()
		return out, nil
	}
	k.sportsTickersMu.RUnlock()

	tickers, err := k.fetchSportsSeriesTickers()
	if err != nil {
		return nil, err
	}
	k.sportsTickersMu.Lock()
	k.sportsTickers = tickers
	k.sportsTickersAt = time.Now()
	k.sportsTickersMu.Unlock()
	return tickers, nil
}

// fetchSportsSeriesTickers 调用 GET /series，筛选 category=Sports 或 isSportsCategory 的 series，返回其 ticker 列表
func (k *Adapter) fetchSportsSeriesTickers() ([]string, error) {
	// 先试 category=Sports（Kalshi 可能用大写）
	base := strings.TrimSuffix(k.cfg.BaseURL, "/")
	u := base + "/series?category=Sports"
	resp, err := k.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("GET /series 失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /series 非200: %d %s", resp.StatusCode, string(body))
	}
	var list model.KalshiSeriesListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("解析 /series 响应失败: %w", err)
	}
	var tickers []string
	for i := range list.Series {
		s := &list.Series[i]
		if isSportsCategory(s.Category) && strings.TrimSpace(s.Ticker) != "" {
			tickers = append(tickers, strings.TrimSpace(s.Ticker))
		}
	}
	if len(tickers) > 0 {
		k.logger.Infof("Kalshi 从 GET /series?category=Sports 获取到 %d 个体育 series_ticker", len(tickers))
		return tickers, nil
	}
	// 若 category=Sports 无结果，则拉全量 series 再按 category 过滤
	u2 := base + "/series"
	resp2, err := k.httpClient.Get(u2)
	if err != nil {
		return nil, fmt.Errorf("GET /series 全量失败: %w", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		return nil, fmt.Errorf("GET /series 全量非200: %d %s", resp2.StatusCode, string(body))
	}
	var list2 model.KalshiSeriesListResponse
	if err := json.NewDecoder(resp2.Body).Decode(&list2); err != nil {
		return nil, fmt.Errorf("解析 /series 全量响应失败: %w", err)
	}
	for i := range list2.Series {
		s := &list2.Series[i]
		if isSportsCategory(s.Category) && strings.TrimSpace(s.Ticker) != "" {
			tickers = append(tickers, strings.TrimSpace(s.Ticker))
		}
	}
	k.logger.Infof("Kalshi 从 GET /series 全量过滤得到 %d 个体育 series_ticker", len(tickers))
	return tickers, nil
}

// fetchSportsEvents 仅拉取体育类事件并全量返回（先取 series_ticker 列表，再按 ticker 请求并合并）。
// 注意：ticker 多时会在内存中累积全部事件，易触发频繁 GC；同步层对 kalshi+sports 已改用 FetchSportsEventsWithYield 流式落库。
func (k *Adapter) fetchSportsEvents() ([]*model.PlatformRawEvent, error) {
	tickers, err := k.getSportsSeriesTickers()
	if err != nil {
		return nil, fmt.Errorf("获取体育 series_ticker 列表失败: %w", err)
	}
	if len(tickers) == 0 {
		k.logger.Warn("Kalshi 未获取到任何体育 series_ticker，跳过事件拉取")
		return nil, nil
	}
	k.logger.Infof("Kalshi 使用 %d 个体育 series_ticker 拉取事件", len(tickers))

	seen := make(map[string]struct{})
	var rawEvents []*model.PlatformRawEvent
	for _, ticker := range tickers {
		u := fmt.Sprintf("%s/events?with_nested_markets=true&status=open&limit=200&series_ticker=%s",
			strings.TrimSuffix(k.cfg.BaseURL, "/"), url.QueryEscape(ticker))
		apiEvs, err := k.fetchEventsRawByURL(u)
		if err != nil {
			k.logger.Warnf("Kalshi series_ticker=%s 拉取失败: %v，跳过", ticker, err)
			continue
		}
		for i := range apiEvs {
			ev := &apiEvs[i]
			if _, dup := seen[ev.EventTicker]; dup {
				continue
			}
			seen[ev.EventTicker] = struct{}{}
			internal := k.apiEventToKalshiEvent(ev)
			rawEvents = append(rawEvents, &model.PlatformRawEvent{
				Platform: k.GetName(),
				ID:       internal.ID,
				Type:     "sports",
				Data:     internal,
			})
		}
	}
	k.logger.Infof("成功获取Kalshi体育事件共%d条", len(rawEvents))
	return rawEvents, nil
}

// FetchSportsEventsWithYield 按 series_ticker 流式拉取体育事件：每拉完一个 ticker 就调用 yield(batch)，便于调用方即时落库，避免全量缓存在内存导致频繁 GC。
// yield 若返回非 nil 会中止后续拉取并返回该错误。seen 跨 ticker 去重，同一 event_ticker 只会在首个出现的 ticker 中交给 yield。
func (k *Adapter) FetchSportsEventsWithYield(ctx context.Context, yield func(batch []*model.PlatformRawEvent) error) (total int, err error) {
	_ = ctx
	tickers, err := k.getSportsSeriesTickers()
	if err != nil {
		return 0, fmt.Errorf("获取体育 series_ticker 列表失败: %w", err)
	}
	if len(tickers) == 0 {
		k.logger.Warn("Kalshi 未获取到任何体育 series_ticker，跳过事件拉取")
		return 0, nil
	}
	k.logger.Infof("Kalshi 使用 %d 个体育 series_ticker 流式拉取事件（每 ticker 落库）", len(tickers))

	seen := make(map[string]struct{})
	for _, ticker := range tickers {
		u := fmt.Sprintf("%s/events?with_nested_markets=true&status=open&limit=200&series_ticker=%s",
			strings.TrimSuffix(k.cfg.BaseURL, "/"), url.QueryEscape(ticker))
		apiEvs, err := k.fetchEventsRawByURL(u)
		if err != nil {
			k.logger.Warnf("Kalshi series_ticker=%s 拉取失败: %v，跳过", ticker, err)
			continue
		}
		var batch []*model.PlatformRawEvent
		for i := range apiEvs {
			ev := &apiEvs[i]
			if _, dup := seen[ev.EventTicker]; dup {
				continue
			}
			seen[ev.EventTicker] = struct{}{}
			internal := k.apiEventToKalshiEvent(ev)
			batch = append(batch, &model.PlatformRawEvent{
				Platform: k.GetName(),
				ID:       internal.ID,
				Type:     "sports",
				Data:     internal,
			})
		}
		if len(batch) > 0 && yield != nil {
			if err := yield(batch); err != nil {
				return total, err
			}
			total += len(batch)
		}
	}
	k.logger.Infof("Kalshi 体育事件流式拉取完成，共 %d 条", total)
	return total, nil
}

// fetchEventsRawByURL 请求 URL 并返回原始 API 事件列表（用于按 series 合并去重）
func (k *Adapter) fetchEventsRawByURL(eventsURL string) ([]model.KalshiEventApi, error) {
	resp, err := k.httpClient.Get(eventsURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API %d: %s", resp.StatusCode, string(body))
	}
	var apiResp model.KalshiEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	return apiResp.Events, nil
}

// fetchEventsByURL 请求 URL 并转为 PlatformRawEvent（非体育或单次请求用）
func (k *Adapter) fetchEventsByURL(eventsURL string, eventType string) ([]*model.PlatformRawEvent, error) {
	apiEvs, err := k.fetchEventsRawByURL(eventsURL)
	if err != nil {
		return nil, fmt.Errorf("获取Kalshi事件失败: %w", err)
	}
	var rawEvents []*model.PlatformRawEvent
	for i := range apiEvs {
		ev := &apiEvs[i]
		internal := k.apiEventToKalshiEvent(ev)
		t := eventType
		if t == "" {
			t = "sports"
		}
		rawEvents = append(rawEvents, &model.PlatformRawEvent{
			Platform: k.GetName(),
			ID:       internal.ID,
			Type:     t,
			Data:     internal,
		})
	}
	return rawEvents, nil
}

// apiEventToKalshiEvent 将 API 返回的单条 event 转为内部 KalshiEvent（含 YES/NO 合约与价格）
func (k *Adapter) apiEventToKalshiEvent(api *model.KalshiEventApi) *model.KalshiEvent {
	openTime := api.StrikeDate
	closeTime := api.StrikeDate
	status := "closed"
	if len(api.Markets) > 0 {
		m := &api.Markets[0]
		if m.OpenTime != "" {
			openTime = m.OpenTime
		}
		if m.CloseTime != "" {
			closeTime = m.CloseTime
		}
		status = m.Status
	}

	contracts := make([]model.KalshiContract, 0)
	for _, m := range api.Markets {
		// YES 价格：优先 yes_ask_dollars，否则 last_price_dollars
		yesPrice := m.YesAskDollars
		if yesPrice == "" {
			yesPrice = m.LastPriceDollars
		}
		if yesPrice != "" {
			contracts = append(contracts, model.KalshiContract{Name: "YES", Price: yesPrice})
		}
		// NO 价格：优先 no_ask_dollars，否则用 1 - last_price
		noPrice := m.NoAskDollars
		if noPrice == "" && m.LastPriceDollars != "" {
			if v, err := strconv.ParseFloat(m.LastPriceDollars, 64); err == nil {
				noPrice = strconv.FormatFloat(1.0-v, 'f', -1, 64)
			}
		}
		if noPrice != "" {
			contracts = append(contracts, model.KalshiContract{Name: "NO", Price: noPrice})
		}
	}
	if len(contracts) == 0 {
		contracts = append(contracts, model.KalshiContract{Name: "YES", Price: "0"})
		contracts = append(contracts, model.KalshiContract{Name: "NO", Price: "0"})
	}

	return &model.KalshiEvent{
		ID:        api.EventTicker,
		Name:      api.Title,
		Status:    status,
		OpenTime:  openTime,
		CloseTime: closeTime,
		Contracts: contracts,
	}
}

func (k *Adapter) ConvertToDBModel(raw []*model.PlatformRawEvent, platformID uint64) ([]*model.Event, []*model.EventOdds, error) {
	var events []*model.Event
	var odds []*model.EventOdds

	for _, r := range raw {
		kalshiEvent, ok := r.Data.(*model.KalshiEvent)
		if !ok || kalshiEvent == nil {
			k.logger.Warn("RawEvent数据类型错误，跳过")
			continue
		}

		// 1. 转换为Event模型（确定性 event_uuid：platform_id_platform_event_id）
		// 截断超长字段，避免数据库字段超限
		title := k.truncateString(kalshiEvent.Name, 256, "title")
		platformEventID := k.truncateString(kalshiEvent.ID, 128, "platform_event_id")
		eventUUID := fmt.Sprintf("%d_%s", platformID, platformEventID)

		// 修复时间类型：若OpenTime/CloseTime是字符串，解析为time.Time
		startTime := k.parseTimeStr(kalshiEvent.OpenTime, "OpenTime")
		endTime := k.parseTimeStr(kalshiEvent.CloseTime, "CloseTime")

		event := &model.Event{
			EventUUID:       eventUUID, // 补充必填字段
			Title:           title,
			Type:            r.Type,
			PlatformID:      platformID,
			PlatformEventID: platformEventID,
			StartTime:       startTime, // 修复时间类型（字符串→time.Time）
			EndTime:         endTime,   // 修复时间类型
			Options:         k.buildOptions(*kalshiEvent),
			Status:          k.mapStatus(kalshiEvent.Status),
			CreatedAt:       time.Now(), // 补充创建时间
			UpdatedAt:       time.Now(), // 补充更新时间
		}
		events = append(events, event)

		// 2. 转换为EventOdds模型（核心修复：循环构建多赔率，移除错误字段）
		eventOddsList := k.buildEventOdds(event.ID, platformID, *kalshiEvent)
		odds = append(odds, eventOddsList...)
	}

	return events, odds, nil
}

// 核心新增：构建EventOdds列表（适配Contracts多选项，移除错误字段）
func (k *Adapter) buildEventOdds(eventID uint64, platformID uint64, ke model.KalshiEvent) []*model.EventOdds {
	var oddsList []*model.EventOdds

	// 遍历Contracts（Kalshi的赔率选项）
	for _, contract := range ke.Contracts {
		// 生成唯一标识（避免重复入库）
		uniqueKey := fmt.Sprintf("%d_%s_%s", platformID, ke.ID, contract.Name)
		// 截断超长的合约名称
		optionName := k.truncateString(contract.Name, 64, "option_name")

		// 转换价格为float64（兜底处理转换失败）
		price := 0.0
		if contract.Price != "" {
			var err error
			price, err = strconv.ParseFloat(contract.Price, 64)
			if err != nil {
				k.logger.Warnf("转换合约%s价格失败: %v，使用0兜底", contract.Name, err)
				price = 0.0
			}
		}

		// 构建EventOdds（移除Odds/MinBet/CacheLevel，补充必填字段）
		odd := &model.EventOdds{
			EventID:             eventID,   // 补充关联事件ID（必填外键）
			UniqueEventPlatform: uniqueKey, // 补充唯一标识（必填）
			PlatformID:          platformID,
			OptionName:          optionName, // 合约名称作为选项名
			Price:               price,      // 使用解析后的价格（替换原Odds字段）
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
			// 移除：Odds/MinBet/CacheLevel（数据库无这些字段）
		}
		oddsList = append(oddsList, odd)
	}

	// 兜底：若没有合约，构建默认Odds
	if len(oddsList) == 0 {
		uniqueKey := fmt.Sprintf("%d_%s", platformID, ke.ID)
		odd := &model.EventOdds{
			EventID:             eventID,
			UniqueEventPlatform: uniqueKey,
			PlatformID:          platformID,
			OptionName:          k.truncateString("default", 64, "option_name"),
			Price:               0.0,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		oddsList = append(oddsList, odd)
	}

	return oddsList
}

// 保留原有buildOptions逻辑（优化错误处理）
func (k *Adapter) buildOptions(event model.KalshiEvent) datatypes.JSON {
	options := make(map[string]interface{})
	for _, c := range event.Contracts {
		options[c.Name] = "available"
	}

	// 序列化 map 为 JSON 字节数组
	jsonBytes, err := json.Marshal(options)
	if err != nil {
		k.logger.WithError(err).Error("Failed to marshal options to JSON")
		return datatypes.JSON("{}") // 返回空 JSON 对象兜底
	}

	return jsonBytes
}

// 工具函数：截断超长字符串
func (k *Adapter) truncateString(s string, maxLen int, fieldName string) string {
	if len(s) <= maxLen {
		return s
	}
	k.logger.Warnf("字段[%s]超长（长度%d），截断为%d字符：%s", fieldName, len(s), maxLen, s[:50]+"...")
	return s[:maxLen]
}

// 工具函数：解析时间字符串为time.Time（适配Kalshi时间格式）
func (k *Adapter) parseTimeStr(timeStr string, fieldName string) time.Time {
	if timeStr == "" {
		k.logger.Warnf("字段[%s]为空，使用当前时间兜底", fieldName)
		return time.Now()
	}

	// Kalshi常见时间格式（根据实际返回值调整）
	timeFormats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		"2006-01-02 15:04:05", // 常规格式
		"2006-01-02",          // 仅日期
	}

	// 遍历格式尝试解析
	for _, format := range timeFormats {
		parsedTime, err := time.Parse(format, timeStr)
		if err == nil {
			return parsedTime
		}
	}

	// 解析失败兜底
	k.logger.Warnf("解析[%s]失败（值：%s），使用当前时间兜底", fieldName, timeStr)
	return time.Now()
}

// 保留原有mapStatus逻辑
func (k *Adapter) mapStatus(kalshiStatus string) string {
	switch kalshiStatus {
	case "open":
		return "active"
	case "closed":
		return "resolved"
	default:
		return "canceled"
	}
}

// isRequestCanceled 判断是否为超时/取消导致的错误，用于避免对 body.Close() 的重复打错日志
func isRequestCanceled(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "request canceled") || strings.Contains(s, "context canceled")
}

// isSportsCategory 判断 Kalshi 的 category 是否属于体育类（只同步体育时过滤用）
func isSportsCategory(category string) bool {
	s := strings.TrimSpace(strings.ToLower(category))
	if s == "" {
		return false
	}
	// Kalshi 体育类常见为 "Sports"；也接受包含 sport 或常见体育系列名
	if s == "sports" || strings.Contains(s, "sport") {
		return true
	}
	// 常见体育系列/分类（按需扩展）
	sportsKeywords := []string{"nfl", "nba", "mlb", "nhl", "soccer", "football", "basketball", "baseball", "hockey", "ufc", "boxing", "tennis", "golf", "olympics"}
	for _, kw := range sportsKeywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
