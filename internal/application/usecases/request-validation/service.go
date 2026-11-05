// Package requestvalidation validates outbound message requests without sending them.
package requestvalidation

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

type DeliveryKind string

const (
	Reply      DeliveryKind = "reply"
	Push       DeliveryKind = "push"
	Multicast  DeliveryKind = "multicast"
	Narrowcast DeliveryKind = "narrowcast"
	Broadcast  DeliveryKind = "broadcast"
)

type MessageKind string

const (
	Text MessageKind = "text"
	Flex MessageKind = "flex"
)

type FlexCard struct{ AltText, Title, Body string }
type Message struct {
	Kind MessageKind
	Text string
	Flex FlexCard
}
type Request struct {
	Delivery DeliveryKind
	Messages []Message
}
type Violation struct{ Field, Reason string }
type Report struct {
	Valid      bool
	Violations []Violation
}

type Gateway interface {
	Validate(context.Context, Request) (Report, error)
}
type Service struct{ gateway Gateway }

func New(gateway Gateway) (*Service, error) {
	if gateway == nil {
		return nil, errors.New("request validation gateway is required")
	}
	return &Service{gateway: gateway}, nil
}

func (s *Service) Validate(ctx context.Context, request Request) (Report, error) {
	if ctx == nil {
		return Report{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	if err := ValidateRequest(request); err != nil {
		return Report{}, err
	}
	request.Messages = append([]Message(nil), request.Messages...)
	return s.gateway.Validate(ctx, request)
}

func ValidateRequest(request Request) error {
	switch request.Delivery {
	case Reply, Push, Multicast, Narrowcast, Broadcast:
	default:
		return errors.New("unsupported delivery kind")
	}
	if len(request.Messages) == 0 || len(request.Messages) > 5 {
		return errors.New("request requires 1-5 messages")
	}
	for _, message := range request.Messages {
		switch message.Kind {
		case Text:
			if strings.TrimSpace(message.Text) == "" || utf8.RuneCountInString(message.Text) > 5000 {
				return errors.New("message text must be 1-5000 characters")
			}
		case Flex:
			if strings.TrimSpace(message.Flex.AltText) == "" || utf8.RuneCountInString(message.Flex.AltText) > 400 {
				return errors.New("card alternative text must be 1-400 characters")
			}
			if strings.TrimSpace(message.Flex.Title) == "" || strings.TrimSpace(message.Flex.Body) == "" {
				return errors.New("card title and body are required")
			}
			if utf8.RuneCountInString(message.Flex.Title) > 300 || utf8.RuneCountInString(message.Flex.Body) > 2000 {
				return errors.New("card content exceeds limits")
			}
		default:
			return errors.New("unsupported message kind")
		}
	}
	return nil
}
