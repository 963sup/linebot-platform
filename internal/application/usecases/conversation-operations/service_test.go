package conversationoperations

import (
	"context"
	"testing"
	"time"
)

type conversationStub struct{ called string }

func (s *conversationStub) Leave(context.Context, Conversation) error { s.called = "leave"; return nil }
func (s *conversationStub) ShowLoading(context.Context, LoadingIndicator) error {
	s.called = "loading"
	return nil
}
func (s *conversationStub) MarkRead(context.Context, ReadTarget) error { s.called = "read"; return nil }
func (s *conversationStub) MarkReadByToken(context.Context, ReadToken) error {
	s.called = "token"
	return nil
}

func TestConversationOperationsDispatchTypedCommands(t *testing.T) {
	stub := &conversationStub{}
	service, err := New(stub)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		want string
		run  func() error
	}{
		{"leave", func() error { return service.Leave(context.Background(), Conversation{Kind: Group, ID: "group-1"}) }},
		{"loading", func() error {
			return service.ShowLoading(context.Background(), LoadingIndicator{ChatID: "user-1", Duration: 10 * time.Second})
		}},
		{"read", func() error { return service.MarkRead(context.Background(), ReadTarget{UserID: "user-1"}) }},
		{"token", func() error { return service.MarkReadByToken(context.Background(), ReadToken{Token: "opaque-token"}) }},
	}
	for _, tc := range cases {
		if err := tc.run(); err != nil {
			t.Fatalf("%s: %v", tc.want, err)
		}
		if stub.called != tc.want {
			t.Fatalf("called %q, want %q", stub.called, tc.want)
		}
	}
}

func TestInvalidConversationCommandHasNoSideEffect(t *testing.T) {
	stub := &conversationStub{}
	service, _ := New(stub)
	invalid := []func() error{
		func() error { return service.Leave(context.Background(), Conversation{Kind: "user", ID: "user-1"}) },
		func() error { return service.Leave(context.Background(), Conversation{Kind: Room, ID: " room-1"}) },
		func() error {
			return service.ShowLoading(context.Background(), LoadingIndicator{ChatID: "user-1", Duration: 7 * time.Second})
		},
		func() error { return service.MarkRead(context.Background(), ReadTarget{}) },
		func() error { return service.MarkReadByToken(context.Background(), ReadToken{}) },
	}
	for _, run := range invalid {
		if err := run(); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
	if stub.called != "" {
		t.Fatalf("invalid command called %q", stub.called)
	}
}
