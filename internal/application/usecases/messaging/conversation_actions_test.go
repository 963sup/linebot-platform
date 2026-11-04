package messaging

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeActions struct{ called string }

func (f *fakeActions) LeaveGroup(context.Context, string) error { f.called = "leave-group"; return nil }
func (f *fakeActions) LeaveRoom(context.Context, string) error  { f.called = "leave-room"; return nil }
func (f *fakeActions) ShowLoading(context.Context, outbound.LINERequest) error {
	f.called = "loading"
	return nil
}
func (f *fakeActions) MarkMessagesRead(context.Context, outbound.LINERequest) error {
	f.called = "mark-read"
	return nil
}
func (f *fakeActions) MarkMessagesReadByToken(context.Context, outbound.LINERequest) error {
	f.called = "mark-read-token"
	return nil
}

func TestConversationActionsValidateAndDispatch(t *testing.T) {
	if _, err := NewConversationActions(nil); err == nil {
		t.Fatal("nil port allowed")
	}
	f := &fakeActions{}
	s, err := NewConversationActions(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []ConversationActionCommand{
		{Action: "leave-group"},
		{Action: "leave-group", ID: "g", Request: outbound.LINERequest{Value: map[string]any{}}},
		{Action: "loading", Request: outbound.LINERequest{Value: []any{}}},
		{Action: "mark-read", Request: outbound.LINERequest{Value: "invalid"}},
		{Action: "not-an-action"},
	} {
		if err := s.Execute(context.Background(), c); err == nil {
			t.Fatalf("accepted %+v", c)
		}
		if f.called != "" {
			t.Fatal("invalid command caused side effect")
		}
	}
	for _, c := range []ConversationActionCommand{
		{Action: "leave-group", ID: "g"},
		{Action: "leave-room", ID: "r"},
		{Action: "loading", Request: outbound.LINERequest{Value: map[string]any{}}},
		{Action: "mark-read", Request: outbound.LINERequest{Value: map[string]any{}}},
		{Action: "mark-read-token", Request: outbound.LINERequest{Value: map[string]any{}}},
	} {
		if err := s.Execute(context.Background(), c); err != nil {
			t.Fatal(err)
		}
		if f.called != c.Action {
			t.Fatalf("called %s for %s", f.called, c.Action)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Execute(ctx, ConversationActionCommand{Action: "leave-group", ID: "g"}); err == nil {
		t.Fatal("canceled context allowed")
	}
}
