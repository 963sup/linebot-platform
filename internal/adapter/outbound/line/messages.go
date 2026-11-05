package line

import (
	"context"
	"errors"
	"net"
	"net/http"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

type Sender struct {
	client *messaging_api.MessagingApiAPI
}

var _ messagingdelivery.DeliveryGateway = (*Sender)(nil)

func NewSender(client *messaging_api.MessagingApiAPI) (*Sender, error) {
	if client == nil {
		return nil, errors.New("LINE Messaging API client is required")
	}
	return &Sender{client: client}, nil
}

func convert(messages []messagingdelivery.Message) ([]messaging_api.MessageInterface, error) {
	result := make([]messaging_api.MessageInterface, 0, len(messages))
	for _, message := range messages {
		if err := message.Validate(); err != nil {
			return nil, err
		}
		switch message.Kind {
		case messagingdelivery.KindText:
			result = append(result, messaging_api.TextMessage{Text: message.Text})
		case messagingdelivery.KindFlex:
			result = append(result, &messaging_api.FlexMessage{
				AltText: message.Flex.AltText,
				Contents: messaging_api.FlexBubble{
					Body: &messaging_api.FlexBox{
						Layout: messaging_api.FlexBoxLAYOUT_VERTICAL,
						Contents: []messaging_api.FlexComponentInterface{
							&messaging_api.FlexText{Text: message.Flex.Title},
							&messaging_api.FlexText{Text: message.Flex.Body},
						},
					},
				},
			})
		default:
			return nil, errors.New("unsupported LINE message kind")
		}
	}
	return result, nil
}

// clientFor applies the request context to a per-call copy. LINE SDK v8.23.0's
// WithContext mutates the receiver; using it on the shared client would race and
// mix concurrent requests. Do not mutate s.client after construction.
func (s *Sender) clientFor(ctx context.Context) (*messaging_api.MessagingApiAPI, error) {
	if ctx == nil {
		return nil, errors.New("request context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	perCall := *s.client
	return perCall.WithContext(ctx), nil
}

func httpStatus(response *http.Response) int {
	if response == nil {
		return 0
	}
	return response.StatusCode
}

// lineAPIError is the single provider-error boundary for LINE SDK calls. It
// closes response bodies, retains only safe classification data, and replaces
// the raw SDK error with a context sentinel or a generic cause. In particular,
// URLs, response bodies, credentials and recipient identifiers in SDK error
// strings are not reachable through the returned error chain.
func lineAPIError(operation string, response *http.Response, err error) error {
	if err == nil {
		return nil
	}
	status := httpStatus(response)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	safeCause := errors.New("LINE provider request failed")
	var networkError net.Error
	switch {
	case errors.Is(err, context.Canceled):
		safeCause = context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		safeCause = context.DeadlineExceeded
	case errors.As(err, &networkError) && networkError.Timeout():
		safeCause = context.DeadlineExceeded
	}
	return resilience.Wrap("line", operation, status, safeCause)
}

func (s *Sender) Reply(ctx context.Context, replyToken string, messages []messagingdelivery.Message) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	mapped, err := convert(messages)
	if err != nil {
		return err
	}
	response, _, err := client.ReplyMessageWithHttpInfo(&messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages:   mapped,
	})
	if err != nil {
		return lineAPIError("reply", response, err)
	}
	return nil
}

func (s *Sender) Push(ctx context.Context, recipient string, messages []messagingdelivery.Message, retryKey string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	mapped, err := convert(messages)
	if err != nil {
		return err
	}
	response, _, err := client.PushMessageWithHttpInfo(&messaging_api.PushMessageRequest{
		To:       recipient,
		Messages: mapped,
	}, retryKey)
	if err != nil {
		return lineAPIError("push", response, err)
	}
	return nil
}
