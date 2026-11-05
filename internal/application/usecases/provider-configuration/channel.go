package providerconfiguration

import (
	"context"
	"time"
)

// ChannelSnapshot is provider-neutral operator visibility into a configured
// messaging channel. Value is deliberately limited to JSON-compatible data;
// SDK response types remain in the outbound adapter.
type ChannelSnapshot struct{ Value any }

type ChannelReader interface {
	BotInfo(context.Context) (ChannelSnapshot, error)
	WebhookEndpoint(context.Context) (ChannelSnapshot, error)
	MessageQuota(context.Context) (ChannelSnapshot, error)
	MessageQuotaConsumption(context.Context) (ChannelSnapshot, error)
	AggregationUnitUsage(context.Context) (ChannelSnapshot, error)
	AggregationUnitNames(context.Context, string, string) (ChannelSnapshot, error)
	NarrowcastProgress(context.Context, string) (ChannelSnapshot, error)
}

type DeliveryCount struct {
	Status     string
	Succeeded  int64
	Failed     int64
	Targeted   int64
	ObservedAt time.Time
}
