// Package linesession verifies a LINE Login identity for provider-neutral consumers.
package linesession

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUnauthenticated = errors.New("LINE session is not authenticated")
	ErrUnavailable     = errors.New("LINE identity verification is unavailable")
)

type Identity struct {
	Subject     string
	ValidatedAt time.Time
}
type Verifier interface {
	Verify(context.Context, string) (Identity, error)
}
type Service struct{ verifier Verifier }

func New(verifier Verifier) (*Service, error) {
	if verifier == nil {
		return nil, errors.New("LINE session verifier is required")
	}
	return &Service{verifier: verifier}, nil
}
func (s *Service) Verify(ctx context.Context, token string) (Identity, error) {
	return s.verifier.Verify(ctx, token)
}
