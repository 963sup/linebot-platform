package messagingdelivery

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

// NarrowcastRequest carries an audience-filtered delivery document without
// exposing a provider SDK type to the application boundary. The outbound
// adapter owns translation of Payload to the provider wire schema.
type NarrowcastRequest struct {
	Payload any
}

type PhoneDeliveryRequest struct{ Payload any }

type PhoneDeliveryGateway interface {
	PushMessagesByPhone(context.Context, PhoneDeliveryRequest) error
	PushTemplatedMessagesByPhone(context.Context, PhoneDeliveryRequest) error
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

// DeliveryGateway is owned by Messaging Delivery. Implementations translate
// these provider-independent messages at the outbound boundary.
type DeliveryGateway interface {
	Reply(context.Context, string, []Message) error
	Push(context.Context, string, []Message, string) error
	Broadcast(context.Context, []Message, string) error
	Multicast(context.Context, []string, []Message, string) error
}

type NarrowcastGateway interface {
	SendNarrowcast(context.Context, NarrowcastRequest, string) error
}
