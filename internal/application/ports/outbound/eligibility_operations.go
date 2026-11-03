package outbound

import (
	"context"
	"encoding/json"
)

// EligibilityOperations requires a channel entitlement confirmed by the
// inbound operator boundary before any method can be invoked.
type EligibilityOperations interface {
	PushMessagesByPhone(context.Context, json.RawMessage) error
	PushTemplatedMessagesByPhone(context.Context, json.RawMessage) error
	PNPMessageStatistics(context.Context, string) (json.RawMessage, error)
	PNPTemplatedMessageStatistics(context.Context, string) (json.RawMessage, error)
	CreateCoupon(context.Context, json.RawMessage) (json.RawMessage, error)
	CloseCoupon(context.Context, string) error
}
