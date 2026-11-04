package outbound

import (
	"context"
	"encoding/json"
)

// ConversationActionManager groups explicit non-message conversation writes.
type ConversationActionManager interface {
	LeaveGroup(context.Context, string) error
	LeaveRoom(context.Context, string) error
	ShowLoading(context.Context, json.RawMessage) error
	MarkMessagesRead(context.Context, json.RawMessage) error
	MarkMessagesReadByToken(context.Context, json.RawMessage) error
}

type AccountLinkManager interface {
	IssueLinkToken(context.Context, string) (json.RawMessage, error)
}
