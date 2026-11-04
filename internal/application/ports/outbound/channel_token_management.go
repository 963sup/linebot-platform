package outbound

import (
	"context"
	"encoding/json"
)

// ChannelTokenManager is not part of the public webhook or the Messaging
// service. Token material is restricted to a protected operator workflow.
type ChannelTokenManager interface {
	IssueStateless(context.Context, string, string, string, string, string) (json.RawMessage, error)
	ValidKeyIDs(context.Context, string, string) (json.RawMessage, error)
	IssueJWT(context.Context, string, string, string) (json.RawMessage, error)
	VerifyJWT(context.Context, string) (json.RawMessage, error)
	RevokeJWT(context.Context, string, string, string) error
	IssueShortLived(context.Context, string, string, string) (json.RawMessage, error)
	VerifyShortLived(context.Context, string) (json.RawMessage, error)
	RevokeShortLived(context.Context, string) error
}
