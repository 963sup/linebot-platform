package cli

import (
	"context"
	"strings"
	"testing"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/google/uuid"
)

type pushRuntimeStub struct{ calls int }

func (s *pushRuntimeStub) Push(context.Context, string, []messagingdelivery.Message, string) error {
	s.calls++
	return nil
}

func (*pushRuntimeStub) Close() error { return nil }

func TestPushRequiresExplicitSend(t *testing.T) {
	err := run([]string{"push", "--to", "U123", "--text", "hello"}, func(string) string { return "" })
	if err == nil {
		t.Fatal("push must require --send")
	}
}

func TestUUIDv4(t *testing.T) {
	u, err := uuidv4()
	if err != nil {
		t.Fatal(err)
	}
	id, err := uuid.Parse(u)
	if err != nil {
		t.Fatalf("invalid UUID %q: %v", u, err)
	}
	if id.Version() != 4 {
		t.Fatalf("UUID version = %d, want 4", id.Version())
	}
}

func TestPushRejectsInvalidRetryKey(t *testing.T) {
	err := run([]string{"push", "--to", "U123", "--text", "hello", "--send", "--retry-key", "invalid-key"}, func(string) string { return "" })
	if err == nil || !strings.Contains(err.Error(), "UUID v4") {
		t.Fatalf("expected UUID v4 validation error, got %v", err)
	}
}

func TestPushUsesInvocationDependencies(t *testing.T) {
	runtime := &pushRuntimeStub{}
	deps := Dependencies{PushRuntime: func(token string) (PushRuntime, error) {
		if token != "test-token" {
			t.Fatalf("token = %q, want test-token", token)
		}
		return runtime, nil
	}}
	err := run(
		[]string{"push", "--to", "U123", "--text", "hello", "--send", "--retry-key", "b547a1a1-e5cb-4f98-83ae-8e571d6ea242"},
		func(string) string { return "test-token" },
		deps,
	)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.calls != 1 {
		t.Fatalf("push calls = %d, want 1", runtime.calls)
	}
}
