package kalshi

import (
	"ForecastSync/internal/config"
	"ForecastSync/internal/utils/httpclient"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	eventsURL := fmt.Sprintf("%s/markets?type=%s", k.cfg.BaseURL, eventType)
	resp, err := k.httpClient.Get(eventsURL)
	if err != nil {
		return nil, fmt.Errorf("获取Kalshi事件失败: %w", err)
	}
	defer resp.Body.Close()

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

		// 1. 转换为Event模型
		event := &model.Event{
			Title:           kalshiEvent.Name,
			Type:            r.Type,
			PlatformID:      platformID,
			PlatformEventID: kalshiEvent.ID,
			StartTime:       kalshiEvent.OpenTime,
			EndTime:         kalshiEvent.CloseTime,
			Options:         k.buildOptions(kalshiEvent),
			Status:          k.mapStatus(kalshiEvent.Status),
		}
		events = append(events, event)

		// 2. 转换为EventOdds模型
		odd := &model.EventOdds{
			PlatformID: platformID,
			Odds:       k.buildOdds(kalshiEvent),
			MinBet:     k.cfg.MinBet,
			CacheLevel: "db",
			UpdatedAt:  time.Now(),
		}
		odds = append(odds, odd)
	}

	return events, odds, nil
}

func (k *Adapter) buildOptions(event model.KalshiEvent) datatypes.JSON {
	options := make(map[string]interface{})
	for _, c := range event.Contracts {
		options[c.Name] = "available"
	}

	// 关键步骤：将 map 序列化为 JSON 字节数组
	jsonBytes, err := json.Marshal(options)
	if err != nil {
		// 错误处理（根据你的业务调整，比如打日志+返回空值）
		k.logger.WithError(err).Error("Failed to marshal options to JSON")
		return datatypes.JSON("{}") // 返回空 JSON 对象兜底
	}

	// 转为 datatypes.JSON（本质是 []byte）
	return jsonBytes
}

// 修复 buildOdds：逻辑和 buildOptions 一致
func (k *Adapter) buildOdds(event model.KalshiEvent) datatypes.JSON {
	odds := make(map[string]interface{})
	for _, c := range event.Contracts {
		odds[c.Name] = c.Price
	}

	// 序列化 map 为 JSON 字节数组
	jsonBytes, err := json.Marshal(odds)
	if err != nil {
		k.logger.WithError(err).Error("Failed to marshal odds to JSON")
		return datatypes.JSON("{}")
	}

	return jsonBytes
}

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
