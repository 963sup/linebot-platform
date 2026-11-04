package messaging

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeSender struct {
	replyCalls int
	pushCalls  int
}

func (f *fakeSender) Reply(_ context.Context, _ string, _ []outbound.Message) error {
	f.replyCalls++
	return nil
}

func (f *fakeSender) Push(_ context.Context, _ string, _ []outbound.Message, _ string) error {
	f.pushCalls++
	return nil
}

func TestServiceValidation(t *testing.T) {
	f := &fakeSender{}
	s, err := NewService(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(nil); err == nil {
		t.Fatal("expected error for nil sender")
	}
	good := []outbound.Message{{Kind: outbound.KindText, Text: "test"}}
	if err := s.Reply(context.Background(), "", good); err == nil {
		t.Fatal("empty reply token should fail")
	}
	if err := s.Reply(context.Background(), "r", nil); err == nil {
		t.Fatal("empty messages should fail")
	}
	if err := s.Push(context.Background(), "U123", good, ""); err == nil {
		t.Fatal("push requires stable retry key")
	}
	if err := s.Reply(context.Background(), "r", good); err != nil {
		t.Fatal(err)
	}
	if err := s.Push(context.Background(), "U123", good, "33333333-3333-4333-8333-333333333333"); err != nil {
		t.Fatal(err)
	}
	if f.replyCalls != 1 || f.pushCalls != 1 {
		t.Fatalf("calls reply=%d push=%d", f.replyCalls, f.pushCalls)
	}
}
