package circle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultSandboxURL Circle 测试环境
	DefaultSandboxURL = "https://api-sandbox.circle.com"
	// DefaultProductionURL Circle 生产环境
	DefaultProductionURL = "https://api.circle.com"
)

// Client Circle API 客户端，用于链资产转 USD
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logrus.Logger
}

// Config Circle 客户端配置
type Config struct {
	BaseURL string
	APIKey  string
	Timeout int // 秒
	Proxy   string
}

// NewClient 创建 Circle 客户端
func NewClient(cfg Config, logger *logrus.Logger) *Client {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultSandboxURL
	}
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if cfg.Proxy != "" {
		if proxyURL, err := url.Parse(cfg.Proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	timeout := 30 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		logger: logger,
	}
}

// exchangeRateRequest Get quote 请求体
type exchangeRateRequest struct {
	From           exchangeAmount `json:"from"`
	To             exchangeAmount `json:"to"`
	IdempotencyKey string         `json:"idempotencyKey"`
	Type           string         `json:"type"`
}

type exchangeAmount struct {
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency"`
}

// exchangeRateResponse Get quote 响应
type exchangeRateResponse struct {
	Data struct {
		ID     string         `json:"id"`
		Rate   float64        `json:"rate"`
		From   exchangeAmount `json:"from"`
		To     exchangeAmount `json:"to"`
		Expiry string         `json:"expiry"`
		Type   string         `json:"type"`
	} `json:"data"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// ConvertToUSD 调用 Circle Exchange Quotes API，将链资产转为 USD
// 支持 USDC/USDT（按 USDC 处理）、USD 直接返回
func (c *Client) ConvertToUSD(ctx context.Context, amount float64, currency string) (float64, error) {
	currency = strings.ToUpper(currency)
	if currency == "USD" {
		return amount, nil
	}
	// USDT 按 USDC 处理（Circle 支持 USDC）
	if currency == "USDT" {
		currency = "USDC"
	}
	if currency != "USDC" {
		return 0, fmt.Errorf("Circle API 暂仅支持 USDC/USDT 转 USD，当前币种: %s", currency)
	}
	if c.apiKey == "" {
		return 0, fmt.Errorf("Circle API key 未配置")
	}

	reqBody := exchangeRateRequest{
		From: exchangeAmount{
			Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
			Currency: currency,
		},
		To: exchangeAmount{
			Currency: "USD",
		},
		IdempotencyKey: uuid.New().String(),
		Type:           "reference",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/exchange/quotes", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Warn("Circle ConvertToUSD HTTP 请求失败")
		return 0, fmt.Errorf("Circle API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result exchangeRateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		c.logger.WithError(err).WithField("body", string(respBody)).Warn("Circle 响应解析失败")
		return 0, fmt.Errorf("Circle API 响应解析失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		msg := result.Message
		if msg == "" {
			msg = string(respBody)
		}
		c.logger.WithField("status", resp.StatusCode).WithField("message", msg).Warn("Circle API 错误")
		return 0, fmt.Errorf("Circle API 错误 %d: %s", resp.StatusCode, msg)
	}

	usdAmount, err := strconv.ParseFloat(result.Data.To.Amount, 64)
	if err != nil {
		return 0, fmt.Errorf("Circle 返回 USD 金额解析失败: %w", err)
	}
	c.logger.WithField("from", amount).WithField("currency", currency).WithField("usd", usdAmount).Debug("Circle ConvertToUSD 成功")
	return usdAmount, nil
}

// ConvertFromUSD 调用 Circle Exchange Quotes API，将 USD 转为目标链资产（如 USDC）
func (c *Client) ConvertFromUSD(ctx context.Context, amountUSD float64, toCurrency string) (float64, error) {
	toCurrency = strings.ToUpper(toCurrency)
	if toCurrency != "USDC" && toCurrency != "USDT" {
		return 0, fmt.Errorf("Circle API 暂仅支持 USD 转 USDC/USDT，当前: %s", toCurrency)
	}
	if c.apiKey == "" {
		return 0, fmt.Errorf("Circle API key 未配置")
	}
	reqBody := exchangeRateRequest{
		From: exchangeAmount{
			Amount:   strconv.FormatFloat(amountUSD, 'f', -1, 64),
			Currency: "USD",
		},
		To: exchangeAmount{
			Currency: toCurrency,
		},
		IdempotencyKey: uuid.New().String(),
		Type:           "reference",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/exchange/quotes", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("Circle API 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result exchangeRateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("Circle API 响应解析失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		msg := result.Message
		if msg == "" {
			msg = string(respBody)
		}
		return 0, fmt.Errorf("Circle API 错误 %d: %s", resp.StatusCode, msg)
	}
	outAmount, err := strconv.ParseFloat(result.Data.To.Amount, 64)
	if err != nil {
		return 0, fmt.Errorf("Circle 返回金额解析失败: %w", err)
	}
	return outAmount, nil
}

// Ping 检查 Circle 服务连通性
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Circle ping 非 200: %d", resp.StatusCode)
	}
	return nil
}
