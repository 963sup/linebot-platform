// Package resilience defines common provider failure semantics without
// implementing generic automatic retries.
package resilience

import (
	"context"
	"errors"
	"fmt"
	"net"
)

type Kind string

const (
	KindCanceled    Kind = "canceled"
	KindTimeout     Kind = "timeout"
	KindUnauthorized Kind = "unauthorized"
	KindRateLimited Kind = "rate_limited"
	KindUnavailable Kind = "unavailable"
	KindUnknown     Kind = "unknown"
)

// ExternalError exposes safe metadata while preserving the original error
// through Unwrap for controlled inspection. Never log the unwrapped raw SDK
// response/body or echo it to end users.
type ExternalError struct {
	Provider       string
	Operation      string
	Kind           Kind
	StatusCode     int
	OutcomeUnknown bool
	cause          error
}

func (e *ExternalError) Error() string {
	return fmt.Sprintf("%s %s failed (%s)", e.Provider, e.Operation, e.Kind)
}

func (e *ExternalError) Unwrap() error { return e.cause }

// Wrap classifies a failed external operation using its HTTP response if any.
// A nil response leaves the remote outcome uncertain even if the caller has
// canceled: non-idempotent operations must not be blindly retried.
func Wrap(provider, operation string, status int, err error) error {
	if err == nil {
		return nil
	}
	kind := KindUnknown
	var netErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		kind = KindCanceled
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		kind = KindTimeout
	case status == 401 || status == 403:
		kind = KindUnauthorized
	case status == 429:
		kind = KindRateLimited
	case status >= 500 && status <= 599:
		kind = KindUnavailable
	}
	return &ExternalError{
		Provider: provider, Operation: operation, Kind: kind,
		StatusCode: status, OutcomeUnknown: status == 0, cause: err,
	}
}
