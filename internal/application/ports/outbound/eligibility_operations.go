package outbound

import "context"

// EligibilityOperations requires a channel entitlement confirmed by the
// inbound operator boundary before any method can be invoked.
type EligibilityOperations interface {
	PushMessagesByPhone(context.Context, LINERequest) error
	PushTemplatedMessagesByPhone(context.Context, LINERequest) error
	PNPMessageStatistics(context.Context, string) (LINEResult, error)
	PNPTemplatedMessageStatistics(context.Context, string) (LINEResult, error)
	CreateCoupon(context.Context, LINERequest) (LINEResult, error)
	CloseCoupon(context.Context, string) error
}
