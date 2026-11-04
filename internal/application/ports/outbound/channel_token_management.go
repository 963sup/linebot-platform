package outbound

import (
	"context"
)

// ChannelTokenManager is not part of the public webhook or the Messaging
// service. Token material is restricted to a protected operator workflow.
type ChannelTokenManager interface {
	IssueStateless(context.Context, string, string, string, string, string) (LINEResult, error)
	ValidKeyIDs(context.Context, string, string) (LINEResult, error)
	IssueJWT(context.Context, string, string, string) (LINEResult, error)
	VerifyJWT(context.Context, string) (LINEResult, error)
	RevokeJWT(context.Context, string, string, string) error
	IssueShortLived(context.Context, string, string, string) (LINEResult, error)
	VerifyShortLived(context.Context, string) (LINEResult, error)
	RevokeShortLived(context.Context, string) error
}
