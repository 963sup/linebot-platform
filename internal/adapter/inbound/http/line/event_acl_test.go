package line

import (
	"testing"

	channelconnection "github.com/963sup/linebot-platform/internal/application/usecases/channel-connection"
	conversation "github.com/963sup/linebot-platform/internal/application/usecases/conversation-ingestion"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func TestLINEEventACLMapsDirectGroupAndRoom(t *testing.T) {
	scope := channelconnection.ResolvedScope{TenantID: "tenant-a", ConnectionID: "line-main"}
	tests := []struct {
		source  webhook.SourceInterface
		kind    conversation.AddressKind
		address string
	}{
		{webhook.UserSource{UserId: "U-user"}, conversation.AddressDirect, "U-user"},
		{webhook.GroupSource{GroupId: "C-group", UserId: "U-user"}, conversation.AddressGroup, "C-group"},
		{webhook.RoomSource{RoomId: "R-room", UserId: "U-user"}, conversation.AddressRoom, "R-room"},
	}
	for _, test := range tests {
		event, err := inboundTextEvent(scope, webhook.MessageEvent{Source: test.source, WebhookEventId: "event-1", Timestamp: 1}, webhook.TextMessageContent{Text: "hello"})
		if err != nil || event.Address.Kind != test.kind || event.Address.ProviderReference != test.address {
			t.Fatalf("source mapping failed: event=%#v err=%v", event, err)
		}
		delivery := deliveryContext(event)
		if err := delivery.Validate(); err != nil || delivery.ConversationReferenceSHA256 == test.address {
			t.Fatalf("unsafe delivery context: %#v err=%v", delivery, err)
		}
	}
}
