package messaging

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeActions struct{ called string }

func (f *fakeActions) LeaveGroup(context.Context, string) error { f.called = "leave-group"; return nil }
func (f *fakeActions) LeaveRoom(context.Context, string) error  { f.called = "leave-room"; return nil }
func (f *fakeActions) ShowLoading(context.Context, json.RawMessage) error {
	f.called = "loading"
	return nil
}
func (f *fakeActions) MarkMessagesRead(context.Context, json.RawMessage) error {
	f.called = "mark-read"
	return nil
}
func (f *fakeActions) MarkMessagesReadByToken(context.Context, json.RawMessage) error {
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
		{Action: "leave-group", ID: "g", Payload: json.RawMessage("{}")},
		{Action: "loading", Payload: json.RawMessage("[]")},
		{Action: "mark-read", Payload: json.RawMessage("invalid")},
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
		{Action: "loading", Payload: json.RawMessage("{}")},
		{Action: "mark-read", Payload: json.RawMessage("{}")},
		{Action: "mark-read-token", Payload: json.RawMessage("{}")},
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
