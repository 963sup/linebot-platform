// Package observability instruments outbound SDK requests without recording
// URLs, authorization, user IDs, request bodies or provider error payloads.
package observability

import (
	"log/slog"
	"net/http"
	"time"
)

type loggingTransport struct {
	provider string
	next     http.RoundTripper
	logger   *slog.Logger
}

// WrapHTTP is an optional safe telemetry decorator for owned SDK transports.
// It does not retry requests, inspect request bodies or alter headers.
func WrapHTTP(provider string, logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		if logger == nil {
			return next
		}
		return &loggingTransport{provider: provider, next: next, logger: logger}
	}
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	started := time.Now()
	resp, err := t.next.RoundTrip(req)
	code := 0
	if resp != nil {
		code = resp.StatusCode
	}
	t.logger.LogAttrs(req.Context(), slog.LevelDebug, "SDK outbound HTTP",
		slog.String("provider", t.provider),
		slog.Int("status_code", code),
		slog.Duration("duration", time.Since(started)),
		slog.Bool("transport_error", err != nil),
	)
	return resp, err
}
