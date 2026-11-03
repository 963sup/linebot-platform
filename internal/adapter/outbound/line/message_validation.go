package line

import (
	"context"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) validate(ctx context.Context, messages []outbound.Message, kind string) error {
	client, err := s.clientFor(ctx)
	if err != nil { return err }
	mapped, err := convert(messages)
	if err != nil { return err }
	req := &messaging_api.ValidateMessageRequest{Messages: mapped}
	switch kind {
	case "reply":
		response, _, err := client.ValidateReplyWithHttpInfo(req)
		return lineWriteError("validateReply",response,err)
	case "push":
		response, _, err := client.ValidatePushWithHttpInfo(req)
		return lineWriteError("validatePush",response,err)
	case "multicast":
		response, _, err := client.ValidateMulticastWithHttpInfo(req)
		return lineWriteError("validateMulticast",response,err)
	case "narrowcast":
		response, _, err := client.ValidateNarrowcastWithHttpInfo(req)
		return lineWriteError("validateNarrowcast",response,err)
	default:
		response, _, err := client.ValidateBroadcastWithHttpInfo(req)
		return lineWriteError("validateBroadcast",response,err)
	}
}

func (s *Sender) ValidateReply(ctx context.Context, messages []outbound.Message) error {
	return s.validate(ctx,messages,"reply")
}
func (s *Sender) ValidatePush(ctx context.Context, messages []outbound.Message) error {
	return s.validate(ctx,messages,"push")
}
func (s *Sender) ValidateMulticast(ctx context.Context, messages []outbound.Message) error {
	return s.validate(ctx,messages,"multicast")
}
func (s *Sender) ValidateNarrowcast(ctx context.Context, messages []outbound.Message) error {
	return s.validate(ctx,messages,"narrowcast")
}
func (s *Sender) ValidateBroadcast(ctx context.Context, messages []outbound.Message) error {
	return s.validate(ctx,messages,"broadcast")
}
