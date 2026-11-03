package outbound

import (
	"context"
	"encoding/json"
)

// MembershipReader reads restricted membership and coupon metadata.
// Authorization and regional eligibility must be checked before enabling a CLI call.
type MembershipReader interface {
	MembershipList(context.Context) (json.RawMessage, error)
	MembershipSubscription(ctx context.Context, userID string) (json.RawMessage, error)
	MembershipUsers(ctx context.Context, membershipID int32, start string, limit int32) (json.RawMessage, error)
	CouponList(ctx context.Context, start string, limit int32) (json.RawMessage, error)
	CouponDetail(ctx context.Context, couponID string) (json.RawMessage, error)
}
