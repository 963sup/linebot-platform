package messaging

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// Conversation actions are operator-only mutations. The public webhook does
// not expose these methods.
type ConversationActionCommand struct {
	Action, ID string
	Request    outbound.LINERequest
}

type ConversationActions struct {
	manager outbound.ConversationActionManager
}

func NewConversationActions(m outbound.ConversationActionManager) (*ConversationActions, error) {
	if m == nil {
		return nil, errors.New("conversation action manager is required")
	}
	return &ConversationActions{manager: m}, nil
}

func ValidateConversationAction(c ConversationActionCommand) error {
	if c.ID != strings.TrimSpace(c.ID) {
		return errors.New("invalid conversation identifier")
	}
	switch c.Action {
	case "leave-group", "leave-room":
		if c.ID == "" || c.Request.Value != nil {
			return errors.New("leave requires an ID and no request")
		}
	case "loading", "mark-read", "mark-read-token":
		if _, ok := c.Request.Value.(map[string]any); c.ID != "" || !ok {
			return errors.New("action requires an object and no ID")
		}
	default:
		return errors.New("unsupported conversation action")
	}
	return nil
}

func (s *ConversationActions) Execute(ctx context.Context, c ConversationActionCommand) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateConversationAction(c); err != nil {
		return err
	}
	switch c.Action {
	case "leave-group":
		return s.manager.LeaveGroup(ctx, c.ID)
	case "leave-room":
		return s.manager.LeaveRoom(ctx, c.ID)
	case "loading":
		return s.manager.ShowLoading(ctx, c.Request)
	case "mark-read":
		return s.manager.MarkMessagesRead(ctx, c.Request)
	case "mark-read-token":
		return s.manager.MarkMessagesReadByToken(ctx, c.Request)
	default:
		return errors.New("unsupported conversation action")
	}
}
