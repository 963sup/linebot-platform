package outbound

import "context"

type WebhookEndpointTest struct {
	Success bool
	StatusCode int32
	Reason string
}

// WebhookConfigurer is an operator-only command boundary.
// It must never be wired to the public webhook runtime.
type WebhookConfigurer interface {
	SetWebhookEndpoint(context.Context, string) error
	TestWebhookEndpoint(context.Context, string) (WebhookEndpointTest, error)
}
