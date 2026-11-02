package messaging

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type Service struct {
	sender outbound.Messaging
}

func NewService(sender outbound.Messaging) (*Service, error) {
	if sender == nil {
		return nil, errors.New("messaging sender is required")
	}
	return &Service{sender: sender}, nil
}

func validate(messages []outbound.Message) error {
	if len(messages) == 0 || len(messages) > 5 {
		return errors.New("LINE requires 1–5 messages per request")
	}
	for _, msg := range messages {
		if err := msg.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Reply(ctx context.Context, replyToken string, messages []outbound.Message) error {
	if strings.TrimSpace(replyToken) == "" {
		return errors.New("reply token is required")
	}
	if err := validate(messages); err != nil {
		return err
	}
	return s.sender.Reply(ctx, replyToken, messages)
}

// Push requires a stable retryKey. If the caller retries an uncertain result,
// it MUST reuse the same retryKey and unchanged message payload.
func (s *Service) Push(ctx context.Context, recipient string, messages []outbound.Message, retryKey string) error {
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(retryKey) == "" {
		return errors.New("recipient and stable retry key are required")
	}
	if err := validate(messages); err != nil {
		return err
	}
	return s.sender.Push(ctx, recipient, messages, retryKey)
}
