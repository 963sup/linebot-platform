package engagement

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type MenuCommand struct {
	Action     string
	RichMenuID string
	UserID     string
	AliasID    string
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
	default:
		return s.manager.DeleteRichMenuAlias(ctx, cmd.AliasID)
	}
}
