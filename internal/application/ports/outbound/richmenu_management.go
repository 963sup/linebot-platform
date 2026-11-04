package outbound

import (
	"context"
	"encoding/json"
)

// RichMenuManager is an explicit command boundary; there is no dynamic SDK call.
type RichMenuManager interface {
	SetDefaultRichMenu(ctx context.Context, richMenuID string) error
	CancelDefaultRichMenu(ctx context.Context) error
	LinkRichMenuToUser(ctx context.Context, userID, richMenuID string) error
	UnlinkRichMenuFromUser(ctx context.Context, userID string) error
	DeleteRichMenu(ctx context.Context, richMenuID string) error
	DeleteRichMenuAlias(ctx context.Context, aliasID string) error
	CreateRichMenu(ctx context.Context, request json.RawMessage) error
	ValidateRichMenu(ctx context.Context, request json.RawMessage) error
	CreateRichMenuAlias(ctx context.Context, request json.RawMessage) error
	UpdateRichMenuAlias(ctx context.Context, aliasID string, request json.RawMessage) error
	BulkLinkRichMenu(ctx context.Context, request json.RawMessage) error
	BulkUnlinkRichMenu(ctx context.Context, request json.RawMessage) error
	RichMenuBatch(ctx context.Context, request json.RawMessage) error
	ValidateRichMenuBatch(ctx context.Context, request json.RawMessage) error
}
