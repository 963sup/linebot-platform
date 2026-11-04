package line

import (
	"context"
	"encoding/json"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) Broadcast(ctx context.Context, messages []outbound.Message, retryKey string) error {
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

func (s *Sender) Multicast(ctx context.Context, recipients []string, messages []outbound.Message, retryKey string) error {
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

func (s *Sender) Narrowcast(ctx context.Context, payload outbound.LINERequest, retryKey string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.NarrowcastRequest
	encoded, err := json.Marshal(payload.Value)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, &request); err != nil {
		return err
	}
	response, _, err := client.NarrowcastWithHttpInfo(&request, retryKey)
	return lineWriteError("narrowcast", response, err)
}
