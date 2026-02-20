package service

import (
	"context"
	"strings"

	"ForecastSync/internal/circle"
)

// FiatConversionService 法币兑换服务（如 Circle），将 USDC/USDT/ETH 转为 USD
// 仅当选中 Kalshi 下单前调用
type FiatConversionService interface {
	// ConvertToUSD 将指定币种金额转为 USD
	ConvertToUSD(ctx context.Context, amount float64, currency string) (usdAmount float64, err error)
}

// NoopFiatConversion 占位实现：直接返回原金额，不做实际兑换（未配置 Circle 时使用）
type NoopFiatConversion struct{}

func NewNoopFiatConversion() *NoopFiatConversion {
	return &NoopFiatConversion{}
}

func (n *NoopFiatConversion) ConvertToUSD(ctx context.Context, amount float64, currency string) (float64, error) {
	_ = ctx
	if strings.ToUpper(currency) == "USDC" || strings.ToUpper(currency) == "USDT" || strings.ToUpper(currency) == "USD" {
		return amount, nil
	}
	return amount, nil
}

// CircleFiatConversion 调用 Circle 测试/生产环境完成链资产转 USD
type CircleFiatConversion struct {
	client *circle.Client
}

// NewCircleFiatConversion 创建 Circle 兑换服务
func NewCircleFiatConversion(client *circle.Client) *CircleFiatConversion {
	return &CircleFiatConversion{client: client}
}

func (c *CircleFiatConversion) ConvertToUSD(ctx context.Context, amount float64, currency string) (float64, error) {
	return c.client.ConvertToUSD(ctx, amount, currency)
}
