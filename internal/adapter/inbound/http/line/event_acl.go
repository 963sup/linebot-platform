package line

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	conversation "github.com/963sup/linebot-platform/internal/application/usecases/conversation-ingestion"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func deliveryContext(event conversation.InboundTextEvent) messagingdelivery.DeliveryContext {
	return messagingdelivery.DeliveryContext{
		ConversationKind:            messagingdelivery.ConversationKind(event.Address.Kind),
		ConversationReferenceSHA256: fingerprint(event.Address.ProviderReference),
		ActorSHA256:                 fingerprint(event.Address.ActorReference),
	}
}

func fingerprint(value string) string {
	if value == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func inboundTextEvent(event webhook.MessageEvent, content webhook.TextMessageContent) (conversation.InboundTextEvent, error) {
	address, err := conversationAddress(event.Source)
	if err != nil {
		return conversation.InboundTextEvent{}, err
	}
	return conversation.NewInboundTextEvent(
		event.WebhookEventId, time.UnixMilli(event.Timestamp), address, content.Text,
	)
}

func conversationAddress(source webhook.SourceInterface) (conversation.Address, error) {
	switch value := source.(type) {
	case webhook.UserSource:
		return conversation.Address{Kind: conversation.AddressDirect, ProviderReference: value.UserId, ActorReference: value.UserId}, nil
	case webhook.GroupSource:
		return conversation.Address{Kind: conversation.AddressGroup, ProviderReference: value.GroupId, ActorReference: value.UserId}, nil
	default:
		return conversation.Address{}, errors.New("unsupported LINE conversation source")
	}
}
