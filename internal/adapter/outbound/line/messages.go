package line

import (
	"context"
	"errors"
	"fmt"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

type Sender struct {
	client *messaging_api.MessagingApiAPI
}

func NewSender(client *messaging_api.MessagingApiAPI) (*Sender, error) {
	if client == nil {
		return nil, errors.New("LINE Messaging API client is required")
	}
	return &Sender{client: client}, nil
}

func convert(messages []outbound.Message) ([]messaging_api.MessageInterface, error) {
	result := make([]messaging_api.MessageInterface, 0, len(messages))
	for _, message := range messages {
		if err := message.Validate(); err != nil {
			return nil, err
		}
		switch message.Kind {
		case outbound.KindText:
			result = append(result, messaging_api.TextMessage{Text: message.Text})
		case outbound.KindFlex:
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

func (s *Sender) Reply(ctx context.Context, replyToken string, messages []outbound.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	mapped, err := convert(messages)
	if err != nil {
		return err
	}
	_, err = s.client.ReplyMessage(&messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages:   mapped,
	})
	if err != nil {
		return fmt.Errorf("LINE reply failed: %w", err)
	}
	return nil
}

func (s *Sender) Push(ctx context.Context, recipient string, messages []outbound.Message, retryKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	mapped, err := convert(messages)
	if err != nil {
		return err
	}
	_, err = s.client.PushMessage(&messaging_api.PushMessageRequest{
		To:       recipient,
		Messages: mapped,
	}, retryKey)
	if err != nil {
		return fmt.Errorf("LINE push failed: %w", err)
	}
	return nil
}
