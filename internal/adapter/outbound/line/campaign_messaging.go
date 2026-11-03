package line

import (
	"context"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) Broadcast(ctx context.Context, messages []outbound.Message, retryKey string) error {
	client, err := s.clientFor(ctx)
	if err != nil { return err }
	mapped, err := convert(messages)
	if err != nil { return err }
	response, _, err := client.BroadcastWithHttpInfo(&messaging_api.BroadcastRequest{Messages: mapped}, retryKey)
	return lineWriteError("broadcast", response, err)
}

func (s *Sender) Multicast(ctx context.Context, recipients []string, messages []outbound.Message, retryKey string) error {
	client, err := s.clientFor(ctx)
	if err != nil { return err }
	mapped, err := convert(messages)
	if err != nil { return err }
	response, _, err := client.MulticastWithHttpInfo(&messaging_api.MulticastRequest{To: recipients, Messages: mapped}, retryKey)
	return lineWriteError("multicast", response, err)
}
