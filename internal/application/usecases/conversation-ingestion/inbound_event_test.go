package conversationingestion

import (
	"testing"
	"time"
)

func TestInboundConversationAddressKinds(t *testing.T) {
	scope := ChannelScope{TenantID: "tenant-a", ConnectionID: "line-main"}
	for _, address := range []Address{
		{Kind: AddressDirect, ProviderReference: "U-user"},
		{Kind: AddressGroup, ProviderReference: "C-group", ActorReference: "U-user"},
		{Kind: AddressRoom, ProviderReference: "R-room", ActorReference: "U-user"},
	} {
		event, err := NewInboundTextEvent(scope, "event-1", time.Now(), address, "hello")
		if err != nil || event.Address.Kind != address.Kind {
			t.Fatalf("address %q rejected: event=%#v err=%v", address.Kind, event, err)
		}
	}
}
