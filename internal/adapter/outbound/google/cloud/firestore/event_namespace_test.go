package firestore

import (
	"context"
	"errors"
	"testing"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEventNamespacePreservesLegacyDedupKey(t *testing.T) {
	saved := legacyEventNamespace{TenantID: "tenant-a", ConnectionID: "line-main", Destination: "bot"}
	namespace, err := eventKeyNamespace("bot", saved, nil)
	if err != nil {
		t.Fatal(err)
	}
	old, _ := messagingdelivery.NewEventKey("tenant-a\x00line-main", "event")
	current, _ := messagingdelivery.NewEventKey(namespace, "event")
	if current != old {
		t.Fatal("cutover changed an existing event key")
	}
}

func TestEventNamespaceDefaultsOnlyWhenLegacyDocumentIsAbsent(t *testing.T) {
	namespace, err := eventKeyNamespace("bot", legacyEventNamespace{}, status.Error(codes.NotFound, "missing"))
	if err != nil || namespace != "bot" {
		t.Fatalf("namespace=%q err=%v", namespace, err)
	}
	for _, readErr := range []error{status.Error(codes.PermissionDenied, "private"), status.Error(codes.Unavailable, "private"), context.Canceled} {
		namespace, err := eventKeyNamespace("bot", legacyEventNamespace{}, readErr)
		if err == nil || namespace != "" {
			t.Fatal("failed lookup silently changed namespace")
		}
	}
	for _, saved := range []legacyEventNamespace{
		{},
		{TenantID: "a", ConnectionID: "b", Destination: "other"},
		{TenantID: "a\x00b", ConnectionID: "c", Destination: "bot"},
	} {
		if _, err := eventKeyNamespace("bot", saved, nil); !errors.Is(err, ErrRepositoryCorrupt) {
			t.Fatalf("bad metadata accepted: %v", err)
		}
	}
}
