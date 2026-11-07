package line

import (
	"testing"

	conversation "github.com/963sup/linebot-platform/internal/application/usecases/conversation-ingestion"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func TestLINEEventACLMapsDirectAndGroup(t *testing.T) {
	tests := []struct {
		source  webhook.SourceInterface
		kind    conversation.AddressKind
		address string
	}{
		{webhook.UserSource{UserId: "U-user"}, conversation.AddressDirect, "U-user"},
		{webhook.GroupSource{GroupId: "C-group", UserId: "U-user"}, conversation.AddressGroup, "C-group"},
	}
	for _, test := range tests {
		event, err := inboundTextEvent(webhook.MessageEvent{Source: test.source, WebhookEventId: "event-1", Timestamp: 1}, webhook.TextMessageContent{Text: "hello"})
		if err != nil || event.Address.Kind != test.kind || event.Address.ProviderReference != test.address {
			t.Fatalf("source mapping failed: event=%#v err=%v", event, err)
		}
		delivery := deliveryContext(event)
		if err := delivery.Validate(); err != nil || delivery.ConversationReferenceSHA256 == test.address {
			t.Fatalf("unsafe delivery context: %#v err=%v", delivery, err)
		}
	}
}

func TestConversationIsolationUsesAddressNotActor(t *testing.T) {
	makeContext := func(source webhook.SourceInterface) string {
		t.Helper()
		event, err := inboundTextEvent(webhook.MessageEvent{Source: source, WebhookEventId: "event", Timestamp: 1}, webhook.TextMessageContent{Text: "hello"})
		if err != nil {
			t.Fatal(err)
		}
		return deliveryContext(event).ConversationReferenceSHA256
	}
	groupA := makeContext(webhook.GroupSource{GroupId: "C-a", UserId: "U-user"})
	groupAOtherActor := makeContext(webhook.GroupSource{GroupId: "C-a", UserId: "U-other"})
	groupB := makeContext(webhook.GroupSource{GroupId: "C-b", UserId: "U-user"})
	direct := makeContext(webhook.UserSource{UserId: "U-user"})
	if groupA != groupAOtherActor || groupA == groupB || groupA == direct {
		t.Fatal("conversation isolation depends on actor or merges distinct sources")
	}
	if _, err := conversationAddress(webhook.RoomSource{RoomId: "R-room"}); err == nil {
		t.Fatal("Room was converted into a supported address")
	}
}
