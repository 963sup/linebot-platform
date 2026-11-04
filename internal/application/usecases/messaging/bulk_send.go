package messaging

import (
	"context"
	"encoding/json"
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

// Narrowcast sends a complete LINE Narrowcast request. The payload remains
// JSON because recipient, demographic filters, and delivery limits are LINE
// API structures that must not be reduced to the application's simple message
// model. LINE's validation endpoint remains available before a live send.
func (s *BulkSender) Narrowcast(ctx context.Context, payload json.RawMessage, retryKey string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if retryKey == "" || retryKey != strings.TrimSpace(retryKey) {
		return errors.New("stable retry key required")
	}
	var request struct {
		Messages json.RawMessage `json:"messages"`
	}
	if len(payload) == 0 || !json.Valid(payload) || json.Unmarshal(payload, &request) != nil {
		return errors.New("valid narrowcast JSON object required")
	}
	var messages []json.RawMessage
	if len(request.Messages) == 0 || json.Unmarshal(request.Messages, &messages) != nil || len(messages) == 0 || len(messages) > 5 {
		return errors.New("narrowcast requires 1-5 messages")
	}
	for _, message := range messages {
		if !json.Valid(message) {
			return errors.New("narrowcast contains invalid message JSON")
		}
		var shape struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(message, &shape) != nil || strings.TrimSpace(shape.Type) == "" {
			return errors.New("narrowcast message type is required")
		}
		if shape.Type == string(outbound.KindText) && strings.TrimSpace(shape.Text) == "" {
			return errors.New("narrowcast text message requires text")
		}
	}
	return s.sender.Narrowcast(ctx, append(json.RawMessage(nil), payload...), retryKey)
}
