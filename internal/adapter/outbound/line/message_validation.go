package line

import (
	"context"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	requestvalidation "github.com/963sup/linebot-platform/internal/application/usecases/request-validation"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) Validate(ctx context.Context, request requestvalidation.Request) (requestvalidation.Report, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return requestvalidation.Report{}, err
	}
	messages := make([]messagingdelivery.Message, 0, len(request.Messages))
	for _, message := range request.Messages {
		mapped := messagingdelivery.Message{Text: message.Text}
		switch message.Kind {
		case requestvalidation.Text:
			mapped.Kind = messagingdelivery.KindText
		case requestvalidation.Flex:
			mapped.Kind = messagingdelivery.KindFlex
			mapped.Flex = messagingdelivery.FlexCard{AltText: message.Flex.AltText, Title: message.Flex.Title, Body: message.Flex.Body}
		}
		messages = append(messages, mapped)
	}
	mapped, err := convert(messages)
	if err != nil {
		return requestvalidation.Report{}, err
	}
	req := &messaging_api.ValidateMessageRequest{Messages: mapped}
	var responseErr error
	switch request.Delivery {
	case requestvalidation.Reply:
		response, _, err := client.ValidateReplyWithHttpInfo(req)
		responseErr = lineWriteError("validateReply", response, err)
	case requestvalidation.Push:
		response, _, err := client.ValidatePushWithHttpInfo(req)
		responseErr = lineWriteError("validatePush", response, err)
	case requestvalidation.Multicast:
		response, _, err := client.ValidateMulticastWithHttpInfo(req)
		responseErr = lineWriteError("validateMulticast", response, err)
	case requestvalidation.Narrowcast:
		response, _, err := client.ValidateNarrowcastWithHttpInfo(req)
		responseErr = lineWriteError("validateNarrowcast", response, err)
	case requestvalidation.Broadcast:
		response, _, err := client.ValidateBroadcastWithHttpInfo(req)
		responseErr = lineWriteError("validateBroadcast", response, err)
	}
	if responseErr != nil {
		return requestvalidation.Report{}, responseErr
	}
	return requestvalidation.Report{Valid: true}, nil
}

var _ requestvalidation.Gateway = (*Sender)(nil)
