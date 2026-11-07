// Package linesession verifies a LINE MINI App identity for provider-neutral consumers.
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
