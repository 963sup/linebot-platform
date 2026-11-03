package outbound

import (
	"context"
	"encoding/json"
)

// RichMenuReader exposes only the seven explicitly required read-only calls.
// Write operations use separate, authorized command contracts.
type RichMenuReader interface {
	RichMenuList(context.Context) (json.RawMessage, error)
	RichMenu(ctx context.Context, menuID string) (json.RawMessage, error)
	DefaultRichMenu(context.Context) (json.RawMessage, error)
	RichMenuAliasList(context.Context) (json.RawMessage, error)
	RichMenuAlias(ctx context.Context, aliasID string) (json.RawMessage, error)
	UserRichMenu(ctx context.Context, userID string) (json.RawMessage, error)
	RichMenuBatchProgress(ctx context.Context, requestID string) (json.RawMessage, error)
}
