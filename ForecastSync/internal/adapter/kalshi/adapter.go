package kalshi

import (
	"ForecastSync/internal/config"
	"ForecastSync/internal/utils/httpclient"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

func (k *Adapter) FetchEvents(ctx context.Context, eventType string) ([]*model.PlatformRawEvent, error) {
	_ = ctx
	// 1. 调用Kalshi API获取事件
	eventsURL := fmt.Sprintf("%s/%s", k.cfg.BaseURL, eventType)
	resp, err := k.httpClient.Get(eventsURL)
	if err != nil {
		return nil, fmt.Errorf("获取Kalshi事件失败: %w", err)
	}
	// 确保响应体关闭，并处理关闭时的错误
	defer func() {
		if err := resp.Body.Close(); err != nil {
			k.logger.Errorf("关闭Kalshi响应体失败: %v", err) // 修正日志格式化错误
		}
	}()

	var kalshiEvents []model.KalshiEvent
	if err := json.NewDecoder(resp.Body).Decode(&kalshiEvents); err != nil {
		return nil, fmt.Errorf("解析Kalshi事件失败: %w", err)
	}

	// 2. 封装为通用RawEvent
	var rawEvents []*model.PlatformRawEvent
	for _, e := range kalshiEvents {
		rawEvents = append(rawEvents, &model.PlatformRawEvent{
			Platform: k.GetName(),
			ID:       e.ID,
			Type:     eventType,
			Data:     e,
		})
	}

	k.logger.Infof("成功获取Kalshi事件共%d条", len(rawEvents))
	return rawEvents, nil
}

func (k *Adapter) ConvertToDBModel(raw []*model.PlatformRawEvent, platformID uint64) ([]*model.Event, []*model.EventOdds, error) {
	var events []*model.Event
	var odds []*model.EventOdds

	for _, r := range raw {
		kalshiEvent, ok := r.Data.(model.KalshiEvent)
		if !ok {
			k.logger.Warn("RawEvent数据类型错误，跳过")
			continue
		}

		// 1. 转换为Event模型（补充必填字段+类型修复+字段截断）
		eventUUID := uuid.New().String() // 补充必填的EventUUID
		// 截断超长字段，避免数据库字段超限
		title := k.truncateString(kalshiEvent.Name, 256, "title")
		platformEventID := k.truncateString(kalshiEvent.ID, 128, "platform_event_id")

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
			Options:         k.buildOptions(kalshiEvent),
			Status:          k.mapStatus(kalshiEvent.Status),
			CreatedAt:       time.Now(), // 补充创建时间
			UpdatedAt:       time.Now(), // 补充更新时间
		}
		events = append(events, event)

		// 2. 转换为EventOdds模型（核心修复：循环构建多赔率，移除错误字段）
		eventOddsList := k.buildEventOdds(event.ID, platformID, kalshiEvent)
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
