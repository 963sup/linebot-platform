package entities_test

import (
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/identity/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

func TestGoogleConnectionLifecycle(t *testing.T) {
	subject, _ := valueobjects.NewGoogleAccountID("google-sub")
	scopes, _ := valueobjects.NewOAuthScopes([]string{"b", "a", "a"})
	connection, err := entities.NewGoogleConnection(subject, scopes)
	if err != nil || !connection.IsActive() {
		t.Fatal("failed to create active connection", err)
	}
	copyScopes := connection.Scopes()
	copyScopes[0] = "tampered"
	if connection.Scopes()[0] == "tampered" {
		t.Fatal("connection leaked scope backing slice")
	}
	changed, err := connection.Disconnect()
	if err != nil || !changed {
		t.Fatal("expected first disconnect", err)
	}
	changed, err = connection.Disconnect()
	if err != nil || changed {
		t.Fatal("disconnect is not idempotent", err)
	}
}

func TestRestoreGoogleConnectionRejectsUnknownState(t *testing.T) {
	subject, _ := valueobjects.NewGoogleAccountID("google-sub")
	scopes, _ := valueobjects.NewOAuthScopes([]string{"scope"})
	if _, err := entities.RestoreGoogleConnection(subject, scopes, "unknown"); err == nil {
		t.Fatal("unknown connection state accepted")
	}
}
