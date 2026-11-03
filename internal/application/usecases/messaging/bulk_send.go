package messaging

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type BulkSender struct{ sender outbound.BulkMessaging }

func NewBulkSender(sender outbound.BulkMessaging) (*BulkSender, error) {
	if sender == nil {
		return nil, errors.New("campaign sender is required")
	}
	return &BulkSender{sender: sender}, nil
}

func validBulkMessages(messages []outbound.Message, retryKey string) error {
	if retryKey == "" || retryKey != strings.TrimSpace(retryKey) {
		return errors.New("stable retry key required")
	}
	if len(messages) == 0 || len(messages) > 5 {
		return errors.New("LINE requires 1-5 messages")
	}
	for _, message := range messages {
		if err := message.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s *BulkSender) Broadcast(ctx context.Context, messages []outbound.Message, retryKey string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validBulkMessages(messages, retryKey); err != nil {
		return err
	}
	return s.sender.Broadcast(ctx, messages, retryKey)
}

func (s *BulkSender) Multicast(ctx context.Context, users []string, messages []outbound.Message, retryKey string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validBulkMessages(messages, retryKey); err != nil {
		return err
	}
	if len(users) == 0 || len(users) > 500 {
		return errors.New("multicast requires 1-500 recipients")
	}
	seen := map[string]bool{}
	for _, user := range users {
		if user == "" || user != strings.TrimSpace(user) || seen[user] {
			return errors.New("invalid or duplicate recipient")
		}
		seen[user] = true
	}
	return s.sender.Multicast(ctx, users, messages, retryKey)
}
