package kalshi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/utils/httpclient"
)

// Ensure Adapter implements interfaces.TradingAdapter
var _ interfaces.TradingAdapter = (*TradingAdapter)(nil)

// TradingAdapter Kalshi 下单适配器，调用配置的 base_url（测试环境 demo-api.kalshi.co 或生产）
type TradingAdapter struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewTradingAdapter 创建 Kalshi 下单适配器
func NewTradingAdapter(cfg *config.Config) *TradingAdapter {
	var platformCfg config.PlatformConfig
	if cfg != nil {
		if k, ok := cfg.Platforms["kalshi"]; ok {
			platformCfg = k
		}
	}
	return &TradingAdapter{
		cfg:        cfg,
		httpClient: httpclient.NewHTTPClient(&platformCfg, nil),
	}
}

// kalshiCreateOrderRequest Kalshi 下单请求体
type kalshiCreateOrderRequest struct {
	Ticker   string `json:"ticker"`
	Side     string `json:"side"`                // yes | no
	Action   string `json:"action"`              // buy | sell
	Count    int    `json:"count"`               // 合约数量
	Type     string `json:"type"`                // limit
	YesPrice int    `json:"yes_price,omitempty"` // 1-99 美分
	NoPrice  int    `json:"no_price,omitempty"`
}

// kalshiCreateOrderResponse Kalshi 下单响应
type kalshiCreateOrderResponse struct {
	Order struct {
		OrderID string `json:"order_id"`
	} `json:"order"`
}

// PlaceOrder 向 Kalshi 测试/生产环境下单
func (t *TradingAdapter) PlaceOrder(ctx context.Context, req *interfaces.PlaceOrderRequest) (platformOrderID string, err error) {
	if req == nil {
		return "", fmt.Errorf("PlaceOrderRequest is nil")
	}
	baseURL := "https://demo-api.kalshi.co/trade-api/v2"
	if t.cfg != nil {
		if k, ok := t.cfg.Platforms["kalshi"]; ok && k.BaseURL != "" {
			baseURL = strings.TrimSuffix(k.BaseURL, "/")
		}
	}
	apiKey := ""
	privateKeyPEM := ""
	if t.cfg != nil {
		if k, ok := t.cfg.Platforms["kalshi"]; ok {
			apiKey = k.AuthKey
			privateKeyPEM = k.AuthSecret
		}
	}
	if apiKey == "" || privateKeyPEM == "" {
		return "", fmt.Errorf("Kalshi API Key 或私钥未配置")
	}

	// Kalshi ticker = platform_event_id（事件下的 market ticker，如 INXD-24DEC31-B4900）
	ticker := req.PlatformEventID
	side := "yes"
	if strings.ToUpper(req.BetOption) == "NO" {
		side = "no"
	}
	// 价格：0-1 转为 1-99 美分
	priceCents := int(math.Round(req.LockedOdds * 100))
	if priceCents < 1 {
		priceCents = 1
	}
	if priceCents > 99 {
		priceCents = 99
	}
	// 数量：USD 金额即合约数（Kalshi 每份合约 $1）
	count := int(req.BetAmount)
	if count < 1 {
		count = 1
	}

	body := kalshiCreateOrderRequest{
		Ticker: ticker,
		Side:   side,
		Action: "buy",
		Count:  count,
		Type:   "limit",
	}
	if side == "yes" {
		body.YesPrice = priceCents
	} else {
		body.NoPrice = priceCents
	}
	bodyBytes, _ := json.Marshal(body)

	path := "/trade-api/v2/portfolio/orders"
	if u, err := url.Parse(baseURL); err == nil && u.Path != "" {
		path = u.Path + "/portfolio/orders"
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature, err := SignRequest(privateKeyPEM, timestamp, "POST", path)
	if err != nil {
		return "", fmt.Errorf("Kalshi 签名失败: %w", err)
	}

	reqURL := baseURL + "/portfolio/orders"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("KALSHI-ACCESS-KEY", apiKey)
	httpReq.Header.Set("KALSHI-ACCESS-TIMESTAMP", timestamp)
	httpReq.Header.Set("KALSHI-ACCESS-SIGNATURE", signature)

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Kalshi 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Kalshi 下单失败 %d: %s", resp.StatusCode, string(respBody))
	}

	var result kalshiCreateOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("Kalshi 响应解析失败: %w", err)
	}
	if result.Order.OrderID == "" {
		return "", fmt.Errorf("Kalshi 返回空 order_id")
	}
	return result.Order.OrderID, nil
}
