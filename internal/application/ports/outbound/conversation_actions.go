package outbound

import "context"

// ConversationActionManager groups explicit non-message conversation writes.
type ConversationActionManager interface {
	LeaveGroup(context.Context, string) error
	LeaveRoom(context.Context, string) error
	ShowLoading(context.Context, LINERequest) error
	MarkMessagesRead(context.Context, LINERequest) error
	MarkMessagesReadByToken(context.Context, LINERequest) error
}

type AccountLinkManager interface {
	IssueLinkTokenResult(context.Context, string) (LINEResult, error)
}
