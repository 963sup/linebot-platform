package messagingdelivery

import (
	"context"
	"errors"
	"strings"
)

type DeliveryService struct{ gateway DeliveryGateway }

// ReplyPolicy maps inbound text to deterministic, provider-independent replies.
type ReplyPolicy struct{ miniAppURL string }

func NewReplyPolicy(miniAppURL string) (ReplyPolicy, error) {
	policy := ReplyPolicy{miniAppURL: miniAppURL}
	if miniAppURL != "" {
		if err := (URIAction{Label: "開啟 MINI App", URI: miniAppURL}).Validate(); err != nil {
			return ReplyPolicy{}, err
		}
	}
	return policy, nil
}

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

// ResponseForText is used by both synchronous and durable inbound adapters.
func (p ReplyPolicy) ResponseForText(text string) Message {
	normalized := strings.ToLower(strings.TrimSpace(text))
	switch normalized {
	case "mini app", "miniapp", "應用", "应用":
		return p.miniAppEntry()
	case "你好", "您好", "嗨", "hi", "hello":
		return Message{Kind: KindText, Text: "你好，輸入「MINI App」可開啟應用，或輸入「功能」查看目前可用指令。"}
	case "功能", "選單", "菜单", "menu", "help", "幫助", "帮助", "?", "？":
		return p.help()
	case "flex", "卡片":
		return Message{Kind: KindFlex, Flex: FlexCard{
			AltText: "LINE Bot 功能卡片", Title: "linebot-platform",
			Body: "Webhook、Reply、Push、Flex Message 基礎架構已準備。",
		}}
	default:
		return Message{Kind: KindText, Text: "請輸入「MINI App」或「功能」。"}
	}
}

func (p ReplyPolicy) miniAppEntry() Message {
	if p.miniAppURL == "" {
		return Message{Kind: KindText, Text: "MINI App 入口尚未設定。"}
	}
	return Message{Kind: KindFlex, Flex: FlexCard{
		AltText: "開啟 LINE MINI App", Title: "LINE MINI App",
		Body:   "開啟 MINI App 使用目前提供的功能。",
		Action: &URIAction{Label: "開啟 MINI App", URI: p.miniAppURL},
	}}
}

func (p ReplyPolicy) help() Message {
	const commands = "可用指令：\n• MINI App\n• 功能"
	if p.miniAppURL == "" {
		return Message{Kind: KindText, Text: commands + "\n\nMINI App 入口尚未設定。"}
	}
	return Message{Kind: KindFlex, Flex: FlexCard{
		AltText: "LINE Bot 功能選單", Title: "LINE Bot",
		Body:   commands,
		Action: &URIAction{Label: "開啟 MINI App", URI: p.miniAppURL},
	}}
}
