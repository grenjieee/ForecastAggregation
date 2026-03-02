package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/utils/httpclient"

	"github.com/GoPolymarket/polymarket-go-sdk"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
)

// Ensure TradingAdapter implements interfaces.TradingAdapter
var _ interfaces.TradingAdapter = (*TradingAdapter)(nil)

// TradingAdapter Polymarket 下单适配器，对接 CLOB API（测试/生产均为 clob.polymarket.com）
type TradingAdapter struct {
	cfg         *config.Config
	gammaClient *http.Client
	clobClient  clob.Client // polymarket CLOB 客户端（接口）
	signer      auth.Signer
}

// gammaEventResponse Gamma API 单事件响应（用于获取 token_id）
type gammaEventResponse struct {
	ID      string        `json:"id"`
	Markets []gammaMarket `json:"markets"`
}

type gammaMarket struct {
	ID                    string  `json:"id"`
	Outcomes              string  `json:"outcomes"`     // "[\"Yes\",\"No\"]" 或 "[\"A\",\"B\"]"
	ClobTokenIds          string  `json:"clobTokenIds"` // "[\"token1\",\"token2\"]"
	OrderPriceMinTickSize float64 `json:"orderPriceMinTickSize"`
	NegRisk               bool    `json:"negRisk"`
	AcceptingOrders       bool    `json:"acceptingOrders"`
}

// NewTradingAdapter 创建 Polymarket 下单适配器
func NewTradingAdapter(cfg *config.Config) *TradingAdapter {
	var platformCfg config.PlatformConfig
	if cfg != nil {
		if p, ok := cfg.Platforms["polymarket"]; ok {
			platformCfg = p
		}
	}
	gammaClient := httpclient.NewHTTPClient(&platformCfg, nil)
	return &TradingAdapter{
		cfg:         cfg,
		gammaClient: gammaClient,
	}
}

// initCLOB 延迟初始化 CLOB 客户端（需私钥与 API 凭证）
func (t *TradingAdapter) initCLOB(ctx context.Context) error {
	if t.clobClient != nil {
		return nil
	}
	var p config.PlatformConfig
	if t.cfg != nil {
		if pp, ok := t.cfg.Platforms["polymarket"]; ok {
			p = pp
		}
	}
	clobBaseURL := "https://clob.polymarket.com"
	if p.ClobBaseURL != "" {
		clobBaseURL = strings.TrimSuffix(p.ClobBaseURL, "/")
	}
	pk := strings.TrimSpace(p.AuthPrivateKey)
	if pk == "" {
		return fmt.Errorf("Polymarket 下单需配置 auth_private_key（私钥）")
	}
	signer, err := auth.NewPrivateKeySigner(pk, 137) // Polygon mainnet
	if err != nil {
		return fmt.Errorf("Polymarket 私钥解析失败: %w", err)
	}
	t.signer = signer

	apiKey := strings.TrimSpace(p.AuthKey)
	secret := strings.TrimSpace(p.AuthSecret)
	passphrase := strings.TrimSpace(p.AuthToken)
	if apiKey == "" || secret == "" || passphrase == "" {
		return fmt.Errorf("Polymarket 下单需配置 auth_key、auth_secret、auth_token（API 凭证，可从私钥 derive 后填入）")
	}
	creds := &auth.APIKey{Key: apiKey, Secret: secret, Passphrase: passphrase}

	cfg := polymarket.DefaultConfig()
	cfg.BaseURLs.CLOB = clobBaseURL
	client := polymarket.NewClient(polymarket.WithConfig(cfg)).WithAuth(signer, creds)
	t.clobClient = client.CLOB
	return nil
}

// resolveTokenID 通过 Gamma API 拉取事件，根据 BetOption 解析出 token_id
func (t *TradingAdapter) resolveTokenID(ctx context.Context, platformEventID string, betOption string) (tokenID string, tickSize float64, negRisk bool, err error) {
	gammaURL := "https://gamma-api.polymarket.com"
	if t.cfg != nil {
		if p, ok := t.cfg.Platforms["polymarket"]; ok && p.BaseURL != "" {
			gammaURL = strings.TrimSuffix(p.BaseURL, "/")
		}
	}
	// 支持 event id 或 slug
	u := gammaURL + "/events/" + url.PathEscape(platformEventID)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", 0, false, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := t.gammaClient.Do(req)
	if err != nil {
		return "", 0, false, fmt.Errorf("请求 Gamma 事件失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", 0, false, fmt.Errorf("Gamma 返回 %d: %s", resp.StatusCode, string(body))
	}
	var ev gammaEventResponse
	if err := json.Unmarshal(body, &ev); err != nil {
		return "", 0, false, fmt.Errorf("解析 Gamma 响应失败: %w", err)
	}
	// 若返回数组（按 slug 查询时），取第一个
	if strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		var arr []gammaEventResponse
		if err := json.Unmarshal(body, &arr); err != nil || len(arr) == 0 {
			return "", 0, false, fmt.Errorf("未找到事件 %s", platformEventID)
		}
		ev = arr[0]
	}
	betOption = strings.TrimSpace(betOption)
	betOptionUpper := strings.ToUpper(betOption)
	isYesNo := betOptionUpper == "YES" || betOptionUpper == "NO"

	for _, m := range ev.Markets {
		outcomes, err := parseJSONStringSlice(m.Outcomes)
		if err != nil || len(outcomes) == 0 {
			continue
		}
		tokens, err := parseJSONStringSlice(m.ClobTokenIds)
		if err != nil || len(tokens) != len(outcomes) {
			continue
		}
		// 二选一市场且选项为 YES/NO：优先按索引取 token（第 1 个=YES，第 2 个=NO），不依赖 outcome 名称
		if len(outcomes) == 2 && isYesNo {
			idx := 0
			if betOptionUpper == "NO" {
				idx = 1
			}
			ts := m.OrderPriceMinTickSize
			if ts <= 0 {
				ts = 0.01
			}
			return strings.TrimSpace(tokens[idx]), ts, m.NegRisk, nil
		}
		// 仅在接受订单的市场中按名称匹配
		if !m.AcceptingOrders {
			continue
		}
		for i, o := range outcomes {
			if strings.EqualFold(strings.TrimSpace(o), betOption) {
				ts := m.OrderPriceMinTickSize
				if ts <= 0 {
					ts = 0.01
				}
				return strings.TrimSpace(tokens[i]), ts, m.NegRisk, nil
			}
		}
	}
	return "", 0, false, fmt.Errorf("事件 %s 中未找到选项 %q 对应的 token", platformEventID, betOption)
}

func parseJSONStringSlice(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PlaceOrder 向 Polymarket CLOB 真实下单（测试环境与生产共用 clob.polymarket.com）
func (t *TradingAdapter) PlaceOrder(ctx context.Context, req *interfaces.PlaceOrderRequest) (platformOrderID string, err error) {
	if req == nil {
		return "", fmt.Errorf("PlaceOrderRequest is nil")
	}
	if err := t.initCLOB(ctx); err != nil {
		return "", err
	}

	tokenID, tickSize, negRisk, err := t.resolveTokenID(ctx, req.PlatformEventID, req.BetOption)
	if err != nil {
		return "", fmt.Errorf("解析 token_id 失败: %w", err)
	}
	// 价格合法性
	price := req.LockedOdds
	if price <= 0 || price >= 1 {
		return "", fmt.Errorf("锁定赔率 %.4f 无效，应在 (0,1) 之间", price)
	}
	// 数量：BUY 侧为 USD 金额
	size := req.BetAmount
	if size < 1 {
		size = 1
	}

	tickStr := fmt.Sprintf("%.4f", tickSize)
	if tickSize >= 0.1 {
		tickStr = fmt.Sprintf("%.1f", tickSize)
	} else if tickSize >= 0.01 {
		tickStr = fmt.Sprintf("%.2f", tickSize)
	} else if tickSize >= 0.001 {
		tickStr = fmt.Sprintf("%.3f", tickSize)
	}
	order, err := clob.NewOrderBuilder(t.clobClient, t.signer).
		TokenID(tokenID).
		Side("BUY").
		Price(price).
		Size(size).
		TickSize(tickStr).
		OrderType(clobtypes.OrderTypeGTC).
		Build()
	if err != nil {
		return "", fmt.Errorf("构建订单失败: %w", err)
	}
	// negRisk 市场需特殊处理，此处先按普通市场；若 SDK 需要可扩展
	_ = negRisk

	resp, err := t.clobClient.CreateOrder(ctx, order)
	if err != nil {
		return "", fmt.Errorf("Polymarket 下单失败: %w", err)
	}
	if resp.ID == "" {
		return "", fmt.Errorf("Polymarket 返回空 order id")
	}
	return resp.ID, nil
}
