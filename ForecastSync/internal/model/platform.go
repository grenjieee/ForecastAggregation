package model

// PlatformRawEvent 所有平台的原始事件通用结构
type PlatformRawEvent struct {
	Platform string      // 平台名称（Polymarket/Kalshi）
	ID       string      // 平台原生事件ID
	Type     string      // 事件类型（sports/politics）
	Data     interface{} // 平台原生数据（PolymarketEvent/KalshiEvent）
}

type PolymarketEvent struct {
	ID               string             `json:"id"`               // 平台事件ID
	Title            string             `json:"title"`            // 事件标题
	Active           bool               `json:"active"`           // 是否激活
	Closed           bool               `json:"closed"`           // 是否关闭
	StartDate        string             `json:"startDate"`        // 开始时间（字符串）
	EndDate          string             `json:"endDate"`          // 结束时间（字符串）
	ResolutionSource string             `json:"resolutionSource"` // 结果来源
	Markets          []PolymarketMarket `json:"markets"`          // 事件对应的盘口/市场（核心：补全Markets字段）
}

type PolymarketOutcome struct {
	Name        string  `json:"name"`        // 选项名称（如"Team A Win"）
	Probability float64 `json:"probability"` // 概率/赔率（对应原Price）
	Liquidity   float64 `json:"liquidity"`   // 流动性（接口返回的字段名）
	Volume      float64 `json:"volume"`      // 交易量（接口返回的字段名）
	// 若接口字段名是其他（如price/odds），对应修改：Price float64 `json:"price"`
}

type PolymarketMarket struct {
	Name          string `json:"name"`          // 盘口名称（如"Win/Lose"）
	Outcomes      string `json:"outcomes"`      // 选项列表（伪JSON数组字符串，如"[\"Team A\",\"Team B\"]"）
	OutcomePrices string `json:"outcomePrices"` // 赔率价格列表（伪JSON数组字符串，如"[\"0.6\",\"0.4\"]"）
}
