package model

import "time"

// PlatformRawEvent 所有平台的原始事件通用结构
type PlatformRawEvent struct {
	Platform string      // 平台名称（Polymarket/Kalshi）
	ID       string      // 平台原生事件ID
	Type     string      // 事件类型（sports/politics）
	Data     interface{} // 平台原生数据（PolymarketEvent/KalshiEvent）
}

type PolymarketEvent struct {
	ID               string             `json:"id"`
	Title            string             `json:"title"`
	StartDate        time.Time          `json:"startDate"`
	EndDate          time.Time          `json:"endDate"`
	ResolutionSource string             `json:"resolutionSource"`
	Active           bool               `json:"active"`
	Closed           bool               `json:"closed"`
	Markets          []PolymarketMarket `json:"markets"`
}

type PolymarketMarket struct {
	ID               string `json:"id"`
	Question         string `json:"question"`
	Outcomes         string `json:"outcomes"` // 伪JSON数组字符串
	OutcomePrices    string `json:"outcomePrices"`
	SportsMarketType string `json:"sportsMarketType"`
}
