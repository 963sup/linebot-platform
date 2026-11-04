package outbound

import (
	"context"
	"encoding/json"
)

// InsightsReader is used by explicit, read-only reporting workflows.
type InsightsReader interface {
	FriendsDemographics(context.Context) (json.RawMessage, error)
	MessageDeliveries(ctx context.Context, date string) (json.RawMessage, error)
	FollowerMetrics(ctx context.Context, date string) (json.RawMessage, error)
	MessageEvent(ctx context.Context, requestID string) (json.RawMessage, error)
	StatisticsPerUnit(ctx context.Context, unit, from, to string) (json.RawMessage, error)
	RichMenuSummary(ctx context.Context, menuID, from, to string) (json.RawMessage, error)
	RichMenuDaily(ctx context.Context, menuID, from, to string) (json.RawMessage, error)
}
