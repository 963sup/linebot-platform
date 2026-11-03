package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"strings"
)

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
	return &ConversationActions{m}, nil
}
func (s *ConversationActions) Execute(ctx context.Context, c ConversationActionCommand) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if strings.TrimSpace(c.Action) == "" {
		return errors.New("action is required")
	}
	return nil
}
