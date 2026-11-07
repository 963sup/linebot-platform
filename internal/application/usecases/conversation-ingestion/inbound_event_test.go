package conversationingestion

import (
	"testing"
	"time"
)

func TestInboundConversationAddressKinds(t *testing.T) {
	for _, address := range []Address{
		{Kind: AddressDirect, ProviderReference: "U-user"},
		{Kind: AddressGroup, ProviderReference: "C-group", ActorReference: "U-user"},
	} {
		event, err := NewInboundTextEvent("event-1", time.Now(), address, "hello")
		if err != nil || event.Address.Kind != address.Kind {
			t.Fatalf("address %q rejected: event=%#v err=%v", address.Kind, event, err)
		}
	}
}

func TestRoomIsNotABusinessAddress(t *testing.T) {
	_, err := NewInboundTextEvent("event", time.Now(), Address{Kind: "room", ProviderReference: "R-room"}, "hello")
	if err == nil {
		t.Fatal("Room is unsupported")
	}
}
