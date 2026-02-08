package model

type KalshiEvent struct {
	ID        string           `json:"id"`        // 平台事件ID
	Name      string           `json:"name"`      // 事件标题
	Status    string           `json:"status"`    // 状态（open/closed）
	OpenTime  string           `json:"openTime"`  // 开始时间（字符串）
	CloseTime string           `json:"closeTime"` // 结束时间（字符串）
	Contracts []KalshiContract `json:"contracts"` // 合约/赔率选项列表
}

// KalshiContract Kalshi合约/赔率选项结构
type KalshiContract struct {
	Name  string `json:"name"`  // 合约名称（如"YES"/"NO"）
	Price string `json:"price"` // 赔率价格（字符串格式，如"0.55"）
}
