package outbound

import "context"

// ChannelReader serves explicit, read-only operations tooling.
// RawMessage retains API-specific report fields without exposing LINE SDK Go types.
// Business use cases should prefer their own narrower, typed DTOs.
type ChannelReader interface {
	BotInfo(context.Context) (LINEResult, error)
	WebhookEndpoint(context.Context) (LINEResult, error)
	MessageQuota(context.Context) (LINEResult, error)
	MessageQuotaConsumption(context.Context) (LINEResult, error)
	AggregationUnitUsage(context.Context) (LINEResult, error)
	AggregationUnitNames(ctx context.Context, limit, start string) (LINEResult, error)
	DeliveryReply(ctx context.Context, date string) (LINEResult, error)
	DeliveryPush(ctx context.Context, date string) (LINEResult, error)
	DeliveryMulticast(ctx context.Context, date string) (LINEResult, error)
	DeliveryBroadcast(ctx context.Context, date string) (LINEResult, error)
	NarrowcastProgress(ctx context.Context, requestID string) (LINEResult, error)
}
