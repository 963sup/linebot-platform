package line

import (
	"context"
	"errors"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) Validate(ctx context.Context, request messagingdelivery.ValidationRequest) (messagingdelivery.ValidationReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return messagingdelivery.ValidationReport{}, err
	}
	mapped, err := convert(request.Messages)
	if err != nil {
		return messagingdelivery.ValidationReport{}, err
	}
	req := &messaging_api.ValidateMessageRequest{Messages: mapped}
	var responseErr error
	switch request.Delivery {
	case messagingdelivery.DeliveryReply:
		response, _, err := client.ValidateReplyWithHttpInfo(req)
		responseErr = lineWriteError("validateReply", response, err)
	case messagingdelivery.DeliveryPush:
		response, _, err := client.ValidatePushWithHttpInfo(req)
		responseErr = lineWriteError("validatePush", response, err)
	case messagingdelivery.DeliveryMulticast:
		response, _, err := client.ValidateMulticastWithHttpInfo(req)
		responseErr = lineWriteError("validateMulticast", response, err)
	case messagingdelivery.DeliveryNarrowcast:
		response, _, err := client.ValidateNarrowcastWithHttpInfo(req)
		responseErr = lineWriteError("validateNarrowcast", response, err)
	case messagingdelivery.DeliveryBroadcast:
		response, _, err := client.ValidateBroadcastWithHttpInfo(req)
		responseErr = lineWriteError("validateBroadcast", response, err)
	default:
		return messagingdelivery.ValidationReport{}, errors.New("unsupported LINE validation delivery kind")
	}
	if responseErr != nil {
		return messagingdelivery.ValidationReport{}, responseErr
	}
	return messagingdelivery.ValidationReport{Valid: true}, nil
}

var _ messagingdelivery.ValidationGateway = (*Sender)(nil)
