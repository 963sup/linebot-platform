package outbound

import "context"

// InsightsReader is used by explicit, read-only reporting workflows.
type InsightsReader interface {
	FriendsDemographics(context.Context) (LINEResult, error)
	MessageDeliveries(ctx context.Context, date string) (LINEResult, error)
	FollowerMetrics(ctx context.Context, date string) (LINEResult, error)
	MessageEvent(ctx context.Context, requestID string) (LINEResult, error)
	StatisticsPerUnit(ctx context.Context, unit, from, to string) (LINEResult, error)
	RichMenuSummary(ctx context.Context, menuID, from, to string) (LINEResult, error)
	RichMenuDaily(ctx context.Context, menuID, from, to string) (LINEResult, error)
}
