package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// Conversation actions are operator-only mutations. The public webhook does
// not expose these methods.
type ConversationActionCommand struct {
	Action, ID string
	Payload    json.RawMessage
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
		if c.ID == "" || len(c.Payload) != 0 {
			return errors.New("leave requires an ID and no JSON request")
		}
	case "loading", "mark-read", "mark-read-token":
		if c.ID != "" || len(c.Payload) == 0 || !json.Valid(c.Payload) ||
			strings.TrimSpace(string(c.Payload))[0] != '{' {
			return errors.New("action requires a JSON object and no ID")
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
		return s.manager.ShowLoading(ctx, c.Payload)
	case "mark-read":
		return s.manager.MarkMessagesRead(ctx, c.Payload)
	case "mark-read-token":
		return s.manager.MarkMessagesReadByToken(ctx, c.Payload)
	default:
		return errors.New("unsupported conversation action")
	}
}
