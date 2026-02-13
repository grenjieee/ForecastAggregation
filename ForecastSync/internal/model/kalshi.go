package model

// KalshiEvent 内部使用的 Kalshi 事件结构（与 DB 转换用）
type KalshiEvent struct {
	ID        string           `json:"id"`        // 平台事件ID（event_ticker）
	Name      string           `json:"name"`      // 事件标题
	Status    string           `json:"status"`    // 状态（open/closed）
	OpenTime  string           `json:"openTime"`  // 开始时间（字符串）
	CloseTime string           `json:"closeTime"` // 结束时间（字符串）
	Contracts []KalshiContract `json:"contracts"` // 合约/赔率选项列表（YES/NO 等）
}

// KalshiContract Kalshi 合约/赔率选项结构
type KalshiContract struct {
	Name  string `json:"name"`  // 合约名称（如 YES / NO）
	Price string `json:"price"` // 赔率价格（字符串格式，如 "0.55"）
}

// ========== Kalshi 官方 API 响应结构（GET /events?with_nested_markets=true） ==========

// KalshiEventsResponse GET /events 的根响应
type KalshiEventsResponse struct {
	Events []KalshiEventApi `json:"events"`
	Cursor string           `json:"cursor"`
}

// KalshiEventApi 单条事件的 API 结构
type KalshiEventApi struct {
	EventTicker  string            `json:"event_ticker"`
	SeriesTicker string            `json:"series_ticker"`
	Title        string            `json:"title"`
	SubTitle     string            `json:"sub_title"`
	Category     string            `json:"category"` // 事件分类：Sports / Politics 等，用于只同步体育时过滤
	StrikeDate   string            `json:"strike_date"`
	Markets      []KalshiMarketApi `json:"markets,omitempty"`
}

// KalshiMarketApi 单条 market 的 API 结构（binary YES/NO）
type KalshiMarketApi struct {
	Ticker           string `json:"ticker"`
	EventTicker      string `json:"event_ticker"`
	Title            string `json:"title"`
	OpenTime         string `json:"open_time"`
	CloseTime        string `json:"close_time"`
	Status           string `json:"status"`
	YesAskDollars    string `json:"yes_ask_dollars"`
	NoAskDollars     string `json:"no_ask_dollars"`
	LastPriceDollars string `json:"last_price_dollars"`
}

// ========== Kalshi GET /series 响应（用于拉取体育类 series_ticker） ==========

// KalshiSeriesListResponse GET /series 的根响应
type KalshiSeriesListResponse struct {
	Series []KalshiSeriesItem `json:"series"`
}

// KalshiSeriesItem 单条 series
type KalshiSeriesItem struct {
	Ticker   string `json:"ticker"`
	Category string `json:"category"`
	Title    string `json:"title"`
}
