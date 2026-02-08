package model

import "time"

// Augur3SportEvent Augur3原始赛事模型（示例，需根据真实API调整）
type Augur3SportEvent struct {
	EventID    string                `json:"event_id"`
	Title      string                `json:"title"`
	SportType  string                `json:"sport_type"`
	State      string                `json:"state"`
	ExpireTime time.Time             `json:"expire_time"`
	Conditions Augur3EventConditions `json:"conditions"`
	Outcomes   []Augur3EventOutcome  `json:"outcomes"`
}

// Augur3EventConditions Augur3原始规则模型
type Augur3EventConditions struct {
	Description string `json:"description"`
	Win         string `json:"win_condition"`
	Loss        string `json:"loss_condition"`
	Draw        string `json:"draw_condition"`
}

// Augur3EventOutcome Augur3原始选项模型
type Augur3EventOutcome struct {
	OutcomeID string  `json:"outcome_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Details   string  `json:"details"`
}
