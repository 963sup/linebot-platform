// Package accountlinktoken owns issuing short-lived, single-use account-link tokens.
package accountlinktoken

import (
	"context"
	"errors"
	"strings"
)

type Subject struct{ ID string }

// Token contains secret material. Callers must not log it and should persist it only
// to an explicitly requested secret destination.
type Token struct{ Value string }

type Gateway interface {
	Issue(context.Context, Subject) (Token, error)
}
type Service struct{ gateway Gateway }

func New(gateway Gateway) (*Service, error) {
	if gateway == nil {
		return nil, errors.New("account-link token gateway is required")
	}
	return &Service{gateway: gateway}, nil
}

func (s *Service) Issue(ctx context.Context, subject Subject) (Token, error) {
	if ctx == nil {
		return Token{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return Token{}, err
	}
	if subject.ID == "" || subject.ID != strings.TrimSpace(subject.ID) {
		return Token{}, errors.New("account subject is required")
	}
	token, err := s.gateway.Issue(ctx, subject)
	if err != nil {
		return Token{}, err
	}
	if token.Value == "" || token.Value != strings.TrimSpace(token.Value) {
		return Token{}, errors.New("account-link provider returned an invalid token")
	}
	return token, nil
}
