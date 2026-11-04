package outbound

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

type MessageKind string

const (
	KindText MessageKind = "text"
	KindFlex MessageKind = "flex"
)

type FlexCard struct {
	AltText string
	Title   string
	Body    string
}

type Message struct {
	Kind MessageKind
	Text string
	Flex FlexCard
}

func (m Message) Validate() error {
	switch m.Kind {
	case KindText:
		if strings.TrimSpace(m.Text) == "" || utf8.RuneCountInString(m.Text) > 5000 {
			return errors.New("message text must be 1–5000 characters")
		}
	case KindFlex:
		if strings.TrimSpace(m.Flex.AltText) == "" || utf8.RuneCountInString(m.Flex.AltText) > 400 {
			return errors.New("Flex alt text must be 1–400 characters")
		}
		if strings.TrimSpace(m.Flex.Title) == "" || strings.TrimSpace(m.Flex.Body) == "" {
			return errors.New("Flex title and body are required")
		}
		if utf8.RuneCountInString(m.Flex.Title) > 300 || utf8.RuneCountInString(m.Flex.Body) > 2000 {
			return errors.New("Flex content exceeds the demo template limit")
		}
	default:
		return errors.New("unsupported message kind")
	}
	return nil
}

// Messaging is a provider-neutral port. Credentials, SDK types and retries belong outside this package.
type Messaging interface {
	Reply(ctx context.Context, replyToken string, messages []Message) error
	Push(ctx context.Context, recipient string, messages []Message, retryKey string) error
}
