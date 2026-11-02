package conversation

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
)

type captureSender struct{ message outbound.Message }

func (c *captureSender) Reply(_ context.Context, _ string, messages []outbound.Message) error {
	c.message = messages[0]
	return nil
}

func (c *captureSender) Push(_ context.Context, _, _ string, _ []outbound.Message, _ string) error {
	return nil
}

func TestResponderTextAndFlex(t *testing.T) {
	f := &captureSender{}
	svc, err := messaging.NewService(f)
	if err != nil {
		t.Fatal(err)
	}
	responder, err := NewResponder(svc)
	if err != nil {
		t.Fatal(err)
	}
	if err := responder.RespondToText(context.Background(), "token", "hello"); err != nil {
		t.Fatal(err)
	}
	if f.message.Kind != outbound.KindText {
		t.Fatalf("expected text response; got %s", f.message.Kind)
	}
	if err := responder.RespondToText(context.Background(), "token", "flex"); err != nil {
		t.Fatal(err)
	}
	if f.message.Kind != outbound.KindFlex {
		t.Fatalf("expected flex response; got %s", f.message.Kind)
	}
}
