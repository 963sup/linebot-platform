package outbound

import "context"

// MembershipReader reads restricted membership metadata.
// Authorization and regional eligibility must be checked before enabling a CLI call.
type MembershipReader interface {
	MembershipList(context.Context) (LINEResult, error)
	MembershipSubscription(ctx context.Context, userID string) (LINEResult, error)
	MembershipUsers(ctx context.Context, membershipID int32, start string, limit int32) (LINEResult, error)
}
