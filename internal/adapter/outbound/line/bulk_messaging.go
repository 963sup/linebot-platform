package line

import (
	"context"
	"encoding/json"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) Broadcast(ctx context.Context, messages []messagingdelivery.Message, retryKey string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	mapped, err := convert(messages)
	if err != nil {
		return err
	}
	response, _, err := client.BroadcastWithHttpInfo(&messaging_api.BroadcastRequest{Messages: mapped}, retryKey)
	return lineWriteError("broadcast", response, err)
}

func (s *Sender) Multicast(ctx context.Context, recipients []string, messages []messagingdelivery.Message, retryKey string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	mapped, err := convert(messages)
	if err != nil {
		return err
	}
	response, _, err := client.MulticastWithHttpInfo(&messaging_api.MulticastRequest{To: recipients, Messages: mapped}, retryKey)
	return lineWriteError("multicast", response, err)
}

func (s *Sender) SendNarrowcast(ctx context.Context, payload messagingdelivery.NarrowcastRequest, retryKey string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.NarrowcastRequest
	encoded, err := json.Marshal(payload.Payload)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, &request); err != nil {
		return err
	}
	response, _, err := client.NarrowcastWithHttpInfo(&request, retryKey)
	return lineWriteError("narrowcast", response, err)
}
