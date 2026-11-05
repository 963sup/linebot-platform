package messagingdelivery

import (
	"context"
	"errors"
	"strings"
)

type DeliveryService struct{ gateway DeliveryGateway }

func NewDeliveryService(gateway DeliveryGateway) (*DeliveryService, error) {
	if gateway == nil {
		return nil, errors.New("delivery gateway is required")
	}
	return &DeliveryService{gateway: gateway}, nil
}

func validateMessages(messages []Message) error {
	if len(messages) == 0 || len(messages) > 5 {
		return errors.New("LINE requires 1–5 messages per request")
	}
	for _, message := range messages {
		if err := message.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeliveryService) Reply(ctx context.Context, replyToken string, messages []Message) error {
	if strings.TrimSpace(replyToken) == "" {
		return errors.New("reply token is required")
	}
	if err := validateMessages(messages); err != nil {
		return err
	}
	return s.gateway.Reply(ctx, replyToken, messages)
}

func (s *DeliveryService) Push(ctx context.Context, recipient string, messages []Message, retryKey string) error {
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(retryKey) == "" {
		return errors.New("recipient and stable retry key are required")
	}
	if err := validateMessages(messages); err != nil {
		return err
	}
	return s.gateway.Push(ctx, recipient, messages, retryKey)
}

func (s *DeliveryService) Broadcast(ctx context.Context, messages []Message, retryKey string) error {
	if strings.TrimSpace(retryKey) == "" {
		return errors.New("stable retry key is required")
	}
	if err := validateMessages(messages); err != nil {
		return err
	}
	return s.gateway.Broadcast(ctx, messages, retryKey)
}

func (s *DeliveryService) Multicast(ctx context.Context, recipients []string, messages []Message, retryKey string) error {
	if len(recipients) == 0 || len(recipients) > 500 || strings.TrimSpace(retryKey) == "" {
		return errors.New("multicast requires 1–500 recipients and stable retry key")
	}
	for _, recipient := range recipients {
		if strings.TrimSpace(recipient) == "" {
			return errors.New("recipient is required")
		}
	}
	if err := validateMessages(messages); err != nil {
		return err
	}
	return s.gateway.Multicast(ctx, recipients, messages, retryKey)
}

func (s *DeliveryService) SendNarrowcast(ctx context.Context, request NarrowcastRequest, retryKey string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if request.Payload == nil || strings.TrimSpace(retryKey) == "" || retryKey != strings.TrimSpace(retryKey) {
		return errors.New("narrowcast payload and stable retry key are required")
	}
	gateway, ok := s.gateway.(NarrowcastGateway)
	if !ok {
		return errors.New("narrowcast delivery is unavailable")
	}
	return gateway.SendNarrowcast(ctx, request, retryKey)
}

// ResponseForText is the application-owned deterministic reply policy used by
// both synchronous and durable inbound adapters.
func ResponseForText(text string) Message {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "flex", "卡片":
		return Message{Kind: KindFlex, Flex: FlexCard{
			AltText: "LINE Bot 功能卡片", Title: "linebot-platform",
			Body: "Webhook、Reply、Push、Flex Message 基礎架構已準備。",
		}}
	default:
		return Message{Kind: KindText, Text: "訊息已收到。輸入 flex 查看 Flex Message 示範。"}
	}
}
