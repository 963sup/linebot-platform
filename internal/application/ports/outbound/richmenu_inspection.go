package outbound

import "context"

// RichMenuReader exposes only the seven explicitly required read-only calls.
// Write operations use separate, authorized command contracts.
type RichMenuReader interface {
	RichMenuList(context.Context) (LINEResult, error)
	RichMenu(ctx context.Context, menuID string) (LINEResult, error)
	DefaultRichMenu(context.Context) (LINEResult, error)
	RichMenuAliasList(context.Context) (LINEResult, error)
	RichMenuAlias(ctx context.Context, aliasID string) (LINEResult, error)
	UserRichMenu(ctx context.Context, userID string) (LINEResult, error)
	RichMenuBatchProgress(ctx context.Context, requestID string) (LINEResult, error)
}
