package bootstrap

import (
	"errors"
	"log/slog"
	"net/http"
)

// NewWebhookRouter preserves the existing handler construction API for tests
// and callers that do not need explicit resource ownership.
// New processes should use NewWebhookRuntime and defer Runtime.Close instead.
func NewWebhookRouter(cfg Config, logger *slog.Logger) (http.Handler, error) {
	if cfg.EventMode == "durable" {
		return nil, errors.New("durable webhook requires NewWebhookRuntime with explicit Close")
	}
	if cfg.LiveLINE() {
		return nil, errors.New("configured webhook requires NewWebhookRuntime with explicit Close")
	}
	runtime, err := NewWebhookRuntime(cfg, logger)
	if err != nil {
		return nil, err
	}
	return runtime.Handler, nil
}
