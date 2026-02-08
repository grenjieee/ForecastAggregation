package model

import "time"

type KalshiMarket struct {
	ID          string           `json:"id"`          // 市场ID
	Ticker      string           `json:"ticker"`      // 市场代码
	Name        string           `json:"name"`        // 市场名称
	Status      string           `json:"status"`      // 状态（open/closed/settled）
	CloseTime   time.Time        `json:"close_time"`  // 关闭时间（对应赛事到期时间）
	Category    string           `json:"category"`    // 分类（对应SportType）
	Description string           `json:"description"` // 规则描述
	Contracts   []KalshiContract `json:"contracts"`   // 合约（对应赛事选项）
}

type KalshiEvent struct {
	ID        string           `json:"id"`         // Kalshi事件ID
	Name      string           `json:"name"`       // 事件名称
	OpenTime  time.Time        `json:"open_time"`  // 开盘时间
	CloseTime time.Time        `json:"close_time"` // 收盘时间
	Status    string           `json:"status"`     // 状态：open/closed/resolved
	Contracts []KalshiContract `json:"contracts"`  // 合约/赔率列表
}

type KalshiContract struct {
	ID          string  `json:"id"`          // 合约ID
	Name        string  `json:"name"`        // 合约名称（Yes/No等）
	Ticker      string  `json:"ticker"`      // 合约代码
	Price       float64 `json:"last_price"`  // 最新价格（对应赔率）
	Description string  `json:"description"` // 合约描述
}

type KalshiSportEvent struct {
	MatchID   string              `json:"match_id"`
	MatchName string              `json:"match_name"`
	Sport     string              `json:"sport"`
	Status    string              `json:"status"`
	EndTime   time.Time           `json:"end_time"`
	Rule      KalshiEventRule     `json:"rule"`
	Options   []KalshiEventOption `json:"options"`
}

type KalshiEventRule struct {
	RuleText string `json:"rule_text"`
	WinRule  string `json:"win_rule"`
	LoseRule string `json:"lose_rule"`
	DrawRule string `json:"draw_rule"`
}

type KalshiEventOption struct {
	OptionID   string  `json:"option_id"`
	OptionName string  `json:"option_name"`
	Odds       float64 `json:"odds"`
	Desc       string  `json:"desc"`
}
