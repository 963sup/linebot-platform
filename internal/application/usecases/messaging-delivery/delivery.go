package messagingdelivery

import (
	"context"
	"errors"
	"strings"
)

type DeliveryService struct{ gateway DeliveryGateway }

// ReplyPolicy maps inbound text to deterministic, provider-independent
// replies. Attendance remains behind its MINI App; this policy only publishes
// the configured entry point.
type ReplyPolicy struct{ attendanceEntryURL string }

func NewReplyPolicy(attendanceEntryURL string) (ReplyPolicy, error) {
	policy := ReplyPolicy{attendanceEntryURL: attendanceEntryURL}
	if attendanceEntryURL != "" {
		if err := (URIAction{Label: "開啟打卡", URI: attendanceEntryURL}).Validate(); err != nil {
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
	case "打卡", "上班", "下班", "出勤", "簽到", "签到", "簽退", "签退":
		return p.attendanceEntry()
	case "班表", "今天班表", "今日班表", "我的班表", "狀態", "打卡狀態", "我的狀態":
		return p.attendanceStatus()
	case "你好", "您好", "嗨", "hi", "hello":
		return Message{Kind: KindText, Text: "你好，我可以協助出勤打卡。輸入「打卡」開啟出勤系統，或輸入「功能」查看目前可用指令。"}
	case "功能", "選單", "菜单", "menu", "help", "幫助", "帮助", "?", "？":
		return p.help()
	case "flex", "卡片":
		return Message{Kind: KindFlex, Flex: FlexCard{
			AltText: "LINE Bot 功能卡片", Title: "linebot-platform",
			Body: "Webhook、Reply、Push、Flex Message 基礎架構已準備。",
		}}
	default:
		return Message{Kind: KindText, Text: "我目前支援出勤相關操作。請輸入「打卡」、「今天班表」、「打卡狀態」或「功能」。"}
	}
}

func (p ReplyPolicy) attendanceEntry() Message {
	if p.attendanceEntryURL == "" {
		return Message{Kind: KindText, Text: "出勤入口尚未設定，請聯絡管理員。輸入「功能」可查看目前支援的指令。"}
	}
	return Message{Kind: KindFlex, Flex: FlexCard{
		AltText: "開啟出勤打卡", Title: "出勤打卡",
		Body:   "開啟 MINI App 完成上班或下班打卡。",
		Action: &URIAction{Label: "開啟打卡", URI: p.attendanceEntryURL},
	}}
}

func (p ReplyPolicy) attendanceStatus() Message {
	const unavailable = "聊天查詢尚未連接班表與打卡紀錄。"
	if p.attendanceEntryURL == "" {
		return Message{Kind: KindText, Text: unavailable + "請聯絡管理員確認出勤狀態。"}
	}
	return Message{Kind: KindFlex, Flex: FlexCard{
		AltText: "查看出勤狀態", Title: "出勤查詢",
		Body:   unavailable + "請開啟出勤系統查看最新資料。",
		Action: &URIAction{Label: "開啟出勤", URI: p.attendanceEntryURL},
	}}
}

func (p ReplyPolicy) help() Message {
	const commands = "可用指令：\n• 打卡／上班／下班\n• 今天班表\n• 打卡狀態\n• 功能"
	if p.attendanceEntryURL == "" {
		return Message{Kind: KindText, Text: commands + "\n\n出勤入口尚未設定，請聯絡管理員。"}
	}
	return Message{Kind: KindFlex, Flex: FlexCard{
		AltText: "LINE Bot 功能選單", Title: "出勤助手",
		Body:   commands,
		Action: &URIAction{Label: "開啟出勤", URI: p.attendanceEntryURL},
	}}
}
