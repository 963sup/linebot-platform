package campaign

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeCampaign struct{ calls int }

func (f *fakeCampaign) Broadcast(context.Context, []outbound.Message, string) error {
	f.calls++
	return nil
}
func (f *fakeCampaign) Multicast(context.Context, []string, []outbound.Message, string) error {
	f.calls++
	return nil
}

func TestCampaignRejectsInvalidRecipientsAndRetryKeys(t *testing.T) {
	f := &fakeCampaign{}
	s, err := NewMessenger(f)
	if err != nil {
		t.Fatal(err)
	}
	messages := []outbound.Message{{Kind: outbound.KindText, Text: "test"}}
	for _, recipients := range [][]string{{}, {"U123", "U123"}, {" U123"}} {
		if err := s.Multicast(context.Background(), recipients, messages, "stable"); err == nil {
			t.Fatalf("invalid recipients: %v", recipients)
		}
	}
	if err := s.Broadcast(context.Background(), messages, ""); err == nil {
		t.Fatal("missing retry key accepted")
	}
	if f.calls != 0 {
		t.Fatal("invalid calls must not reach adapter")
	}
	if err := s.Broadcast(context.Background(), messages, "stable"); err != nil {
		t.Fatal(err)
	}
	if err := s.Multicast(context.Background(), []string{"U123"}, messages, "stable"); err != nil {
		t.Fatal(err)
	}
	if f.calls != 2 {
		t.Fatalf("calls: %d", f.calls)
	}
}
