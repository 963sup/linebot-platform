package outbound

import "context"

// RichMenuManager is an explicit command boundary; there is no dynamic SDK call.
type RichMenuManager interface {
	SetDefaultRichMenu(ctx context.Context, richMenuID string) error
	CancelDefaultRichMenu(ctx context.Context) error
	LinkRichMenuToUser(ctx context.Context, userID, richMenuID string) error
	UnlinkRichMenuFromUser(ctx context.Context, userID string) error
	DeleteRichMenu(ctx context.Context, richMenuID string) error
	DeleteRichMenuAlias(ctx context.Context, aliasID string) error
}
