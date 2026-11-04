package engagement

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type MenuCommand struct {
	Action      string
	RichMenuID  string
	UserID      string
	AliasID     string
	Payload     outbound.LINERequest
	Image       []byte
	ContentType string
}

type MenuManager struct {
	manager outbound.RichMenuManager
}

func NewMenuManager(manager outbound.RichMenuManager) (*MenuManager, error) {
	if manager == nil {
		return nil, errors.New("rich menu manager is required")
	}
	return &MenuManager{manager: manager}, nil
}

func validID(id string) bool {
	return id != "" && id == strings.TrimSpace(id)
}

func ValidateMenuCommand(cmd MenuCommand) error {
	switch cmd.Action {
	case "create", "validate", "create-alias", "bulk-link", "bulk-unlink", "batch", "validate-batch":
		if !isLINEObject(cmd.Payload) || cmd.RichMenuID != "" || cmd.UserID != "" || cmd.AliasID != "" || len(cmd.Image) != 0 || cmd.ContentType != "" {
			return errors.New("a valid JSON payload and no IDs are required")
		}
	case "update-alias":
		if !validID(cmd.AliasID) || !isLINEObject(cmd.Payload) || cmd.RichMenuID != "" || cmd.UserID != "" || len(cmd.Image) != 0 || cmd.ContentType != "" {
			return errors.New("alias ID and a valid JSON payload are required")
		}
	case "upload-image":
		if !validID(cmd.RichMenuID) || len(cmd.Image) == 0 || (cmd.ContentType != "image/png" && cmd.ContentType != "image/jpeg") || cmd.UserID != "" || cmd.AliasID != "" || cmd.Payload.Value != nil {
			return errors.New("rich menu ID, PNG or JPEG image, and no additional input are required")
		}
	case "set-default", "delete-menu":
		if !validID(cmd.RichMenuID) || cmd.UserID != "" || cmd.AliasID != "" {
			return errors.New("valid rich menu ID and no additional IDs are required")
		}
	case "cancel-default":
		if cmd.RichMenuID != "" || cmd.UserID != "" || cmd.AliasID != "" {
			return errors.New("cancel-default accepts no IDs")
		}
	case "link-user":
		if !validID(cmd.RichMenuID) || !validID(cmd.UserID) || cmd.AliasID != "" {
			return errors.New("link-user requires rich menu and user IDs")
		}
	case "unlink-user":
		if !validID(cmd.UserID) || cmd.RichMenuID != "" || cmd.AliasID != "" {
			return errors.New("unlink-user requires only a user ID")
		}
	case "delete-alias":
		if !validID(cmd.AliasID) || cmd.RichMenuID != "" || cmd.UserID != "" {
			return errors.New("delete-alias requires only an alias ID")
		}
	default:
		return errors.New("unsupported rich menu action")
	}
	return nil
}

// Execute requires authorization at the inbound boundary and performs only
// the explicitly specified mutation. Failed or unknown remote outcomes are
// never automatically replayed.
func (s *MenuManager) Execute(ctx context.Context, cmd MenuCommand) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateMenuCommand(cmd); err != nil {
		return err
	}
	switch cmd.Action {
	case "create":
		return s.manager.CreateRichMenu(ctx, cmd.Payload)
	case "validate":
		return s.manager.ValidateRichMenu(ctx, cmd.Payload)
	case "create-alias":
		return s.manager.CreateRichMenuAlias(ctx, cmd.Payload)
	case "update-alias":
		return s.manager.UpdateRichMenuAlias(ctx, cmd.AliasID, cmd.Payload)
	case "bulk-link":
		return s.manager.BulkLinkRichMenu(ctx, cmd.Payload)
	case "bulk-unlink":
		return s.manager.BulkUnlinkRichMenu(ctx, cmd.Payload)
	case "batch":
		return s.manager.RichMenuBatch(ctx, cmd.Payload)
	case "validate-batch":
		return s.manager.ValidateRichMenuBatch(ctx, cmd.Payload)
	case "set-default":
		return s.manager.SetDefaultRichMenu(ctx, cmd.RichMenuID)
	case "cancel-default":
		return s.manager.CancelDefaultRichMenu(ctx)
	case "link-user":
		return s.manager.LinkRichMenuToUser(ctx, cmd.UserID, cmd.RichMenuID)
	case "unlink-user":
		return s.manager.UnlinkRichMenuFromUser(ctx, cmd.UserID)
	case "delete-menu":
		return s.manager.DeleteRichMenu(ctx, cmd.RichMenuID)
	case "delete-alias":
		return s.manager.DeleteRichMenuAlias(ctx, cmd.AliasID)
	default:
		return errors.New("unsupported rich menu action")
	}
}
