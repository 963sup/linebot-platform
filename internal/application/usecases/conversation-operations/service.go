// Package conversationoperations owns explicit non-message conversation mutations.
package conversationoperations

import (
	"context"
	"errors"
	"strings"
	"time"
)

type ConversationKind string

const (
	Group ConversationKind = "group"
	Room  ConversationKind = "room"
)

type Conversation struct {
	Kind ConversationKind
	ID   string
}
type LoadingIndicator struct {
	ChatID   string
	Duration time.Duration
}
type ReadTarget struct{ UserID string }
type ReadToken struct{ Token string }

type Gateway interface {
	Leave(context.Context, Conversation) error
	ShowLoading(context.Context, LoadingIndicator) error
	MarkRead(context.Context, ReadTarget) error
	MarkReadByToken(context.Context, ReadToken) error
}

type Service struct{ gateway Gateway }

func New(gateway Gateway) (*Service, error) {
	if gateway == nil {
		return nil, errors.New("conversation gateway is required")
	}
	return &Service{gateway: gateway}, nil
}

func (s *Service) Leave(ctx context.Context, conversation Conversation) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if conversation.Kind != Group && conversation.Kind != Room {
		return errors.New("conversation must be a group or room")
	}
	if err := validateID(conversation.ID); err != nil {
		return err
	}
	return s.gateway.Leave(ctx, conversation)
}

func (s *Service) ShowLoading(ctx context.Context, indicator LoadingIndicator) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateID(indicator.ChatID); err != nil {
		return err
	}
	if indicator.Duration < 5*time.Second || indicator.Duration > 60*time.Second || indicator.Duration%(5*time.Second) != 0 {
		return errors.New("loading duration must be a multiple of five seconds from 5-60 seconds")
	}
	return s.gateway.ShowLoading(ctx, indicator)
}

func (s *Service) MarkRead(ctx context.Context, target ReadTarget) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateID(target.UserID); err != nil {
		return err
	}
	return s.gateway.MarkRead(ctx, target)
}

func (s *Service) MarkReadByToken(ctx context.Context, token ReadToken) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateID(token.Token); err != nil {
		return errors.New("read token is required")
	}
	return s.gateway.MarkReadByToken(ctx, token)
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func validateID(value string) error {
	if value == "" || value != strings.TrimSpace(value) {
		return errors.New("conversation identifier is required")
	}
	return nil
}
