package interfaces

import "context"

// LiveOddsRow 单条实时赔率（用于下单前选平台与落库）
type LiveOddsRow struct {
	PlatformID uint64
	OptionName string
	Price      float64
}

// LiveOddsFetcher 按平台与平台侧事件 ID 拉取当前赔率（用于下单时实时选平台与事后更新 event_odds）
type LiveOddsFetcher interface {
	FetchLiveOdds(ctx context.Context, platformID uint64, platformEventID string) ([]LiveOddsRow, error)
}
