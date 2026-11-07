package firestore

import (
	"context"
	"errors"
	"testing"
)

func TestNewClientRejectsInvalidInputBeforeCredentials(t *testing.T) {
	if _, err := NewClient(nil, "project"); err == nil {
		t.Fatal("nil context accepted")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewClient(cancelled, "project"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	for _, project := range []string{"", "   "} {
		if _, err := NewClient(context.Background(), project); err == nil {
			t.Fatal("empty project accepted")
		}
	}
}
