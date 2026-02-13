package kalshi

import (
	"context"
	"fmt"

	"ForecastSync/internal/interfaces"
)

// Ensure Adapter implements interfaces.TradingAdapter
var _ interfaces.TradingAdapter = (*TradingAdapter)(nil)

// TradingAdapter Kalshi 下单适配器（当前为 stub，可后续接真实 API）
type TradingAdapter struct{}

// NewTradingAdapter 创建 Kalshi 下单适配器
func NewTradingAdapter() *TradingAdapter {
	return &TradingAdapter{}
}

// PlaceOrder 向 Kalshi 下单，当前返回 stub 订单号
func (t *TradingAdapter) PlaceOrder(ctx context.Context, req *interfaces.PlaceOrderRequest) (platformOrderID string, err error) {
	_ = ctx
	if req == nil {
		return "", fmt.Errorf("PlaceOrderRequest is nil")
	}
	// Stub：真实实现应调用 Kalshi 下单 API
	return "stub_kalshi_" + req.PlatformEventID, nil
}
