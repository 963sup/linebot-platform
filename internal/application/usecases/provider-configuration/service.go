// Package providerconfiguration owns operator-only provider endpoint configuration.
package providerconfiguration

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

type WebhookTestResult struct {
	Reachable  bool
	StatusCode int
	Reason     string
}

type WebhookGateway interface {
	SetEndpoint(context.Context, string) error
	TestEndpoint(context.Context, string) (WebhookTestResult, error)
}

type Service struct{ gateway WebhookGateway }

func New(gateway WebhookGateway) (*Service, error) {
	if gateway == nil {
		return nil, errors.New("webhook gateway is required")
	}
	return &Service{gateway: gateway}, nil
}

func (s *Service) ConfigureWebhook(ctx context.Context, endpoint string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := ValidateWebhookEndpoint(endpoint); err != nil {
		return err
	}
	return s.gateway.SetEndpoint(ctx, endpoint)
}

func (s *Service) TestWebhook(ctx context.Context, endpoint string) (WebhookTestResult, error) {
	if err := validateContext(ctx); err != nil {
		return WebhookTestResult{}, err
	}
	if err := ValidateWebhookEndpoint(endpoint); err != nil {
		return WebhookTestResult{}, err
	}
	return s.gateway.TestEndpoint(ctx, endpoint)
}

func ValidateWebhookEndpoint(raw string) error {
	if raw == "" || raw != strings.TrimSpace(raw) {
		return errors.New("webhook endpoint is required")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Opaque != "" {
		return errors.New("webhook endpoint must be an absolute HTTPS URL without credentials, query, or fragment")
	}
	return nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}
