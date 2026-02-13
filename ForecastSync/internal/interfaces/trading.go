package interfaces

import "context"

// PlaceOrderRequest 下单请求参数
type PlaceOrderRequest struct {
	PlatformID      uint64  // 目标平台 ID
	PlatformEventID string  // 平台侧事件 ID
	BetOption       string  // 下注选项（与 event_odds.option_name 对齐）
	BetAmount       float64 // 下注金额
	LockedOdds      float64 // 锁定赔率
}

// TradingAdapter 各平台下单接口（真实调用平台下单 API）
type TradingAdapter interface {
	// PlaceOrder 向该平台下单，返回平台订单号
	PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (platformOrderID string, err error)
}
