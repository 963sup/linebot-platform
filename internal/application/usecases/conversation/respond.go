package conversation

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
)

// Responder is the deliberately small MVP conversation use case.
// Replace the canned response only when persistent processing is implemented.
type Responder struct {
	messaging *messaging.Service
}

func NewResponder(service *messaging.Service) (*Responder, error) {
	if service == nil {
		return nil, errors.New("messaging service is required")
	}
	return &Responder{messaging: service}, nil
}

func (r *Responder) RespondToText(ctx context.Context, replyToken string, text string) error {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "flex", "卡片":
		return r.messaging.Reply(ctx, replyToken, []outbound.Message{{
			Kind: outbound.KindFlex,
			Flex: outbound.FlexCard{
				AltText: "LINE Bot 功能卡片",
				Title:   "linebot-platform",
				Body:    "Webhook、Reply、Push、Flex Message 基礎架構已準備。",
			},
		}})
	default:
		return r.messaging.Reply(ctx, replyToken, []outbound.Message{{
			Kind: outbound.KindText,
			Text: "訊息已收到。輸入 flex 查看 Flex Message 示範。",
		}})
	}
}
