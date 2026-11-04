package messaging

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeBulkMessaging struct{ calls int }

func (f *fakeBulkMessaging) Broadcast(context.Context, []outbound.Message, string) error {
	f.calls++
	return nil
}
func (f *fakeBulkMessaging) Multicast(context.Context, []string, []outbound.Message, string) error {
	f.calls++
	return nil
}
func (f *fakeBulkMessaging) Narrowcast(context.Context, outbound.LINERequest, string) error {
	f.calls++
	return nil
}

func TestBulkSendNarrowcastValidatesPayloadBeforeCallingAdapter(t *testing.T) {
	f := &fakeBulkMessaging{}
	s, err := NewBulkSender(f)
	if err != nil {
		t.Fatal(err)
	}
	valid := outbound.LINERequest{Value: map[string]any{"messages": []any{map[string]any{"type": "text", "text": "hello"}}, "recipient": map[string]any{"type": "operator", "and": []any{}}}}
	for _, payload := range []outbound.LINERequest{{}, {Value: []any{}}, {Value: map[string]any{"messages": []any{}}}, {Value: map[string]any{"messages": []any{map[string]any{"type": "text"}}}}} {
		if err := s.Narrowcast(context.Background(), payload, "stable"); err == nil {
			t.Fatalf("invalid payload accepted: %#v", payload)
		}
	}
	if f.calls != 0 {
		t.Fatal("invalid narrowcasts must not reach adapter")
	}
	if err := s.Narrowcast(context.Background(), valid, "stable"); err != nil {
		t.Fatal(err)
	}
	if f.calls != 1 {
		t.Fatalf("calls: %d", f.calls)
	}
}

func TestBulkSendRejectsInvalidRecipientsAndRetryKeys(t *testing.T) {
	f := &fakeBulkMessaging{}
	s, err := NewBulkSender(f)
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
