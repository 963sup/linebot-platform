package cli

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

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
