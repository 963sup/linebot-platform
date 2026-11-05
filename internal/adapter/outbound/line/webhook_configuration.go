package line

import (
	"context"
	"errors"

	providerconfiguration "github.com/963sup/linebot-platform/internal/application/usecases/provider-configuration"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) SetWebhookEndpoint(ctx context.Context, endpoint string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	response, _, err := client.SetWebhookEndpointWithHttpInfo(&messaging_api.SetWebhookEndpointRequest{Endpoint: endpoint})
	return lineWriteError("setWebhookEndpoint", response, err)
}

func (s *Sender) TestEndpoint(ctx context.Context, endpoint string) (providerconfiguration.WebhookTestResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.WebhookTestResult{}, err
	}
	response, result, err := client.TestWebhookEndpointWithHttpInfo(&messaging_api.TestWebhookEndpointRequest{Endpoint: endpoint})
	if err != nil {
		return providerconfiguration.WebhookTestResult{}, lineWriteError("testWebhookEndpoint", response, err)
	}
	if result == nil {
		return providerconfiguration.WebhookTestResult{}, errors.New("LINE webhook test response is empty")
	}
	return providerconfiguration.WebhookTestResult{Reachable: result.Success, StatusCode: int(result.StatusCode), Reason: result.Reason}, nil
}

func (s *Sender) SetEndpoint(ctx context.Context, endpoint string) error {
	return s.SetWebhookEndpoint(ctx, endpoint)
}

var _ providerconfiguration.WebhookGateway = (*Sender)(nil)
