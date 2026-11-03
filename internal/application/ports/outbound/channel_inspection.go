package outbound

import (
	"context"
	"encoding/json"
)

// ChannelReader serves explicit, read-only operations tooling.
// RawMessage retains API-specific report fields without exposing LINE SDK Go types.
// Business use cases should prefer their own narrower, typed DTOs.
type ChannelReader interface {
	BotInfo(context.Context) (json.RawMessage, error)
	WebhookEndpoint(context.Context) (json.RawMessage, error)
	MessageQuota(context.Context) (json.RawMessage, error)
	MessageQuotaConsumption(context.Context) (json.RawMessage, error)
	AggregationUnitUsage(context.Context) (json.RawMessage, error)
	AggregationUnitNames(ctx context.Context, limit, start string) (json.RawMessage, error)
	DeliveryReply(ctx context.Context, date string) (json.RawMessage, error)
	DeliveryPush(ctx context.Context, date string) (json.RawMessage, error)
	DeliveryMulticast(ctx context.Context, date string) (json.RawMessage, error)
	DeliveryBroadcast(ctx context.Context, date string) (json.RawMessage, error)
	NarrowcastProgress(ctx context.Context, requestID string) (json.RawMessage, error)
}
