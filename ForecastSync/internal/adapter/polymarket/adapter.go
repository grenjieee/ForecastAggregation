package polymarket

import (
	"ForecastSync/internal/config"
	"ForecastSync/internal/utils/httpclient"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"

	"github.com/google/uuid"
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
	// 确保响应体关闭，并处理关闭时的错误
	defer func() {
		if err := sportsResp.Body.Close(); err != nil {
			p.logger.Errorf("关闭sports响应体失败: %v", err) // 修正日志格式化错误
		}
	}()

	var sports []struct {
		Series string `json:"series"`
		Tags   string `json:"tags"`
	}
	if err := json.NewDecoder(sportsResp.Body).Decode(&sports); err != nil {
		return nil, fmt.Errorf("解析运动列表失败: %w", err)
	}

	// 2. 构造每个运动类别下的赛事 k:tagId v:series_id
	var ballSeries = make(map[string]string, len(sports))
	for _, s := range sports {
		tagSlice := strings.Split(s.Tags, ",")
		for _, tag := range tagSlice {
			ballSeries[tag] = s.Series
		}
	}

	// 3. 爬取每个系列的事件
	var rawEvents []*model.PlatformRawEvent
	for tagId, series := range ballSeries {
		if len(tagId) == 0 || len(series) == 0 {
			continue
		}
		//拼接获取当赛事下的所有赛事预测盘口
		eventsURL := fmt.Sprintf("%s/events?series_id=%s&tag_id=%s&active=true&closed=false&order=startTime&ascending=true",
			p.cfg.BaseURL, series, tagId)
		eventsResp, err := p.httpClient.Get(eventsURL)
		if err != nil {
			p.logger.Warnf("爬取%s事件失败: %v", series, err)
			continue
		}

		// 核心修复：解析响应（兼容数组/单个对象）
		polyEvents, parseErr := p.parsePolymarketEvents(eventsResp, series)
		// 无论解析成功与否，都要关闭响应体（关键：避免资源泄漏）
		if closeErr := eventsResp.Body.Close(); closeErr != nil {
			p.logger.Errorf("关闭%s事件响应体失败: %v", series, closeErr)
		}
		if parseErr != nil {
			p.logger.Warnf("解析%s事件失败: %v", series, parseErr)
			continue
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
	}

	p.logger.Infof("成功获取Polymarket事件共%d条", len(rawEvents))
	return rawEvents, nil
}

func (p *Adapter) parsePolymarketEvents(resp *http.Response, series string) ([]model.PolymarketEvent, error) {
	// 1. 先读取原始响应体（解决Body只能读取一次的问题）
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 2. 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("接口返回非200状态码: %d，响应体：%s", resp.StatusCode, string(rawBody))
	}

	// 3. 空数据兜底
	trimmedBody := strings.TrimSpace(string(rawBody))
	if trimmedBody == "" || trimmedBody == "null" {
		p.logger.Warnf("解析%s事件失败: 响应体为空或null", series)
		return []model.PolymarketEvent{}, nil
	}

	// 4. 先解析为通用接口，判断数据类型
	var rawData interface{}
	if err := json.Unmarshal(rawBody, &rawData); err != nil {
		return nil, fmt.Errorf("解析原始数据失败: %w，响应体：%s", err, trimmedBody)
	}

	var polyEvents []model.PolymarketEvent
	switch v := rawData.(type) {
	case []interface{}:
		// 场景1：返回数组 → 直接解析到切片
		if err := json.Unmarshal(rawBody, &polyEvents); err != nil {
			return nil, fmt.Errorf("解析数组失败: %w", err)
		}
		p.logger.Debugf("解析%s事件成功，共%d条（数组格式）", series, len(polyEvents))

	case map[string]interface{}:
		// 场景2：返回单个对象 → 转为切片
		var singleEvent model.PolymarketEvent
		if err := json.Unmarshal(rawBody, &singleEvent); err != nil {
			return nil, fmt.Errorf("解析单个对象失败: %w", err)
		}
		polyEvents = append(polyEvents, singleEvent)
		p.logger.Debugf("解析%s事件成功，共1条（单个对象格式）", series)

	default:
		// 场景3：未知类型 → 报错
		return nil, fmt.Errorf("未知数据类型: %T，响应体：%s", v, trimmedBody)
	}

	return polyEvents, nil
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

		// 1. 转换为Event模型（补充必填的EventUUID，截断超长字段）
		eventUUID := uuid.New().String() // 生成全局唯一ID（与数据库字段匹配）
		// 截断超长字段，避免数据库字段超限
		title := p.truncateString(polyEvent.Title, 256, "title")
		platformEventID := p.truncateString(polyEvent.ID, 128, "platform_event_id")

		// 核心修复：将字符串时间解析为time.Time类型
		startTime := p.parseTimeStr(polyEvent.StartDate, "StartDate")
		endTime := p.parseTimeStr(polyEvent.EndDate, "EndDate")

		event := &model.Event{
			EventUUID:       eventUUID, // 补充必填字段（数据库表中该字段非空）
			Title:           title,
			Type:            r.Type,
			PlatformID:      platformID,
			PlatformEventID: platformEventID,
			StartTime:       startTime, // 修复：字符串→time.Time
			EndTime:         endTime,   // 修复：字符串→time.Time
			Options:         p.buildOptions(polyEvent),
			Status:          p.mapStatus(polyEvent.Active, polyEvent.Closed),
			ResultSource:    p.truncateResultSource(polyEvent.ResolutionSource), // 截断结果来源
			CreatedAt:       time.Now(),                                         // 补充创建时间
			UpdatedAt:       time.Now(),                                         // 补充更新时间
		}
		events = append(events, event)

		// 2. 转换为EventOdds模型（核心修复：改用buildOdds解析的赔率，移除不存在的Options字段）
		eventOddsList := p.buildEventOdds(event.ID, platformID, polyEvent)
		odds = append(odds, eventOddsList...)
	}

	return events, odds, nil
}

// 核心修改：buildEventOdds - 从Markets/Outcomes解析赔率，放弃不存在的Options字段
func (p *Adapter) buildEventOdds(eventID uint64, platformID uint64, pe model.PolymarketEvent) []*model.EventOdds {
	var oddsList []*model.EventOdds

	// 遍历Markets（你原有代码中解析赔率的核心逻辑）
	for _, market := range pe.Markets {
		// 解析Outcomes（选项名称）和OutcomePrices（赔率价格）
		outcomes, err := parseJSONArrayString(market.Outcomes)
		if err != nil {
			p.logger.Warnf("解析Outcomes失败: %v，跳过该market", err)
			continue
		}
		prices, err := parseJSONArrayString(market.OutcomePrices)
		if err != nil {
			p.logger.Warnf("解析OutcomePrices失败: %v，跳过该market", err)
			continue
		}

		// 遍历每个选项，匹配价格
		for i, outcomeName := range outcomes {
			// 防止索引越界
			if i >= len(prices) {
				p.logger.Warnf("选项%s无对应价格，跳过", outcomeName)
				continue
			}
			// 转换价格为float64
			price, err := strconv.ParseFloat(prices[i], 64)
			if err != nil {
				p.logger.Warnf("转换价格%s失败: %v，跳过", prices[i], err)
				continue
			}

			// 生成唯一标识（避免重复入库）
			uniqueKey := fmt.Sprintf("%d_%s_%s_%s", platformID, pe.ID, market.Name, outcomeName)
			// 截断超长选项名称
			optionName := p.truncateString(outcomeName, 64, "option_name")

			// 构建EventOdds（使用解析后的价格，移除不存在的Liquidity/Volume）
			odd := &model.EventOdds{
				EventID:             eventID,
				UniqueEventPlatform: uniqueKey,
				PlatformID:          platformID,
				OptionName:          optionName,
				Price:               price, // 核心：使用解析后的赔率价格
				UpdatedAt:           time.Now(),
				CreatedAt:           time.Now(),
				// 移除Liquidity/Volume（接口无该字段，数据库设默认值0）
			}
			oddsList = append(oddsList, odd)
		}
	}

	// 兜底：若没有解析到任何赔率，构建默认Odds
	if len(oddsList) == 0 {
		uniqueKey := fmt.Sprintf("%d_%s", platformID, pe.ID)
		odd := &model.EventOdds{
			EventID:             eventID,
			UniqueEventPlatform: uniqueKey,
			PlatformID:          platformID,
			OptionName:          p.truncateString("default", 64, "option_name"),
			Price:               0.0, // 兜底值
			UpdatedAt:           time.Now(),
			CreatedAt:           time.Now(),
		}
		oddsList = append(oddsList, odd)
	}

	return oddsList
}

func (p *Adapter) truncateString(s string, maxLen int, fieldName string) string {
	if len(s) <= maxLen {
		return s
	}
	p.logger.Warnf("字段[%s]超长（长度%d），截断为%d字符：%s", fieldName, len(s), maxLen, s[:50]+"...")
	return s[:maxLen]
}

func (p *Adapter) truncateResultSource(s string) *string {
	if s == "" {
		return nil
	}
	truncated := p.truncateString(s, 256, "result_source")
	return &truncated
}

// 保留你原有buildOptions逻辑（适配Markets/Outcomes）
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

// 保留你原有mapStatus逻辑
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

// parseJSONArrayString 解析伪JSON数组字符串（保留原有逻辑）
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
func (p *Adapter) parseTimeStr(timeStr string, fieldName string) time.Time {
	if timeStr == "" {
		p.logger.Warnf("字段[%s]为空，使用当前时间兜底", fieldName)
		return time.Now()
	}

	// 定义Polymarket接口可能返回的时间格式（根据实际返回值调整！）
	timeFormats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"（最常见的ISO格式）
		"2006-01-02 15:04:05", // 常规年月日时分秒
		"2006-01-02",          // 仅日期
		"2006/01/02",          // 斜杠分隔的日期
		time.RFC1123,          // 兼容HTTP头时间格式
	}

	// 遍历格式尝试解析
	for _, format := range timeFormats {
		parsedTime, err := time.Parse(format, timeStr)
		if err == nil {
			return parsedTime
		}
	}

	// 所有格式解析失败，兜底返回当前时间并记录详细日志
	p.logger.Warnf("解析[%s]失败（值：%s），支持的格式：%v，使用当前时间兜底", fieldName, timeStr, timeFormats)
	return time.Now()
}
