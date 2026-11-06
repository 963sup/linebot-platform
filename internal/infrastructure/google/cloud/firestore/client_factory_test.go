package firestore

import (
	"context"
	"testing"
)

func TestNewClientWithDatabaseRejectsInvalidInputBeforeCredentials(t *testing.T) {
	if _, err := NewClientWithDatabase(nil, "project", DefaultDatabase); err == nil {
		t.Fatal("nil context accepted")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewClientWithDatabase(cancelled, "project", DefaultDatabase); err == nil {
		t.Fatal("cancelled context accepted")
	}
	for _, input := range []struct{ project, database string }{
		{"", DefaultDatabase},
		{"project", ""},
		{"project", " database"},
		{"project", "database/path"},
	} {
		if _, err := NewClientWithDatabase(context.Background(), input.project, input.database); err == nil {
			t.Fatalf("invalid input accepted: %#v", input)
		}
	}
}
