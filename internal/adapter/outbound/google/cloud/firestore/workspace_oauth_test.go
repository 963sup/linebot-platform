package firestore

import (
	"context"
	"errors"
	"strings"
	"testing"

	identity "github.com/963sup/linebot-platform/internal/application/usecases/identity"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOAuthDocumentKeysAreOpaqueAndScoped(t *testing.T) {
	id, err := connectionID("line-user-1", "google-subject-1")
	if err != nil {
		t.Fatal(err)
	}
	id2, err := connectionID("line-user-2", "google-subject-1")
	if err != nil {
		t.Fatal(err)
	}
	if id == id2 || len(id) != 64 || strings.Contains(id, "line-user") {
		t.Fatal("OAuth connection document leaked or mixed identities")
	}
	if _, err := connectionID("", "subject"); err == nil {
		t.Fatal("empty owner accepted")
	}
	if _, err := connectionID("line", ""); err == nil {
		t.Fatal("empty subject accepted")
	}
	state := strings.Repeat("A", 43)
	if _, err := opaqueStateID(state); err != nil {
		t.Fatal(err)
	}
	if _, err := opaqueStateID("not-state"); err == nil {
		t.Fatal("invalid state accepted")
	}
}

func TestOAuthRepositoryReadErrorClassification(t *testing.T) {
	ctx := context.Background()
	if err := oauthSessionReadError(ctx, status.Error(codes.NotFound, "missing")); !errors.Is(err, identity.ErrOAuthSessionNotFound) {
		t.Fatalf("session not found = %v", err)
	}
	if err := oauthConnectionReadError(ctx, status.Error(codes.NotFound, "missing")); !errors.Is(err, identity.ErrOAuthConnectionNotFound) {
		t.Fatalf("connection not found = %v", err)
	}
	if err := oauthSessionReadError(ctx, status.Error(codes.Unavailable, "outage")); errors.Is(err, identity.ErrOAuthSessionNotFound) {
		t.Fatalf("session outage misclassified: %v", err)
	}
	if err := oauthConnectionReadError(ctx, status.Error(codes.Unavailable, "outage")); errors.Is(err, identity.ErrOAuthConnectionNotFound) {
		t.Fatalf("connection outage misclassified: %v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := oauthSessionReadError(canceled, status.Error(codes.NotFound, "missing")); !errors.Is(err, context.Canceled) {
		t.Fatalf("context did not win: %v", err)
	}
}
