package firestore

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	identity "github.com/963sup/linebot-platform/internal/application/usecases/identity"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	oauth "github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
)

func TestLegacyConnectionJSONCompatibility(t *testing.T) {
	const legacyJSON = `{"OwnerID":"line-123","GoogleSubject":"google-456","Scopes":["scope-a","scope-b"],"Credentials":{"AccessToken":"access","RefreshToken":"refresh","TokenType":"Bearer","Expiry":"2026-11-01T00:00:00Z"},"Version":7,"Active":true}`
	var record legacyOAuthConnection
	if err := json.Unmarshal([]byte(legacyJSON), &record); err != nil {
		t.Fatal(err)
	}
	auth, err := authorizationFromLegacy(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(auth.Link.PendingEvents()) != 0 || auth.Link.Owner().String() != "line-123" ||
		auth.Link.GoogleAccount().String() != "google-456" || !auth.Link.IsActive() {
		t.Fatal("legacy record did not restore identity without spurious events")
	}
	next, err := legacyFromAuthorization(auth)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(next)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != legacyJSON {
		t.Fatalf("legacy JSON wire format changed: %s", encoded)
	}

	record.Active = false
	disabled, err := authorizationFromLegacy(record)
	if err != nil || disabled.Link.IsActive() {
		t.Fatal("inactive record incorrectly restored", err)
	}
	if _, err := authorizationFromLegacy(legacyOAuthConnection{OwnerID: "", GoogleSubject: "sub", Scopes: []string{"scope"}, Active: true}); err == nil {
		t.Fatal("invalid identity accepted")
	}
}

func TestStoredCiphertextRemainsDecodableWithoutMigration(t *testing.T) {
	keys, err := oauth.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{0x42}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	link, err := factories.NewVerifiedLink("line-1", "google-sub", []string{"scope-a"})
	if err != nil {
		t.Fatal(err)
	}
	auth := identity.OAuthAuthorization{
		Link: link, Version: 9,
		Credentials: identity.OAuthCredentials{
			AccessToken: "private-access", RefreshToken: "private-refresh",
			TokenType: "Bearer", Expiry: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	store := &OAuthStore{keys: keys}
	id, err := connectionID("line-1", "google-sub")
	if err != nil {
		t.Fatal(err)
	}
	keyID, ciphertext, err := store.encodeConnection(id, auth)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := store.decodeConnection(id, storedConnection{
		Ciphertext: ciphertext, KeyID: keyID, Active: true, Version: 9,
	})
	if err != nil || decoded.Link.GoogleAccount() != auth.Link.GoogleAccount() ||
		decoded.Credentials.RefreshToken != "private-refresh" || decoded.Version != 9 {
		t.Fatal("legacy ciphertext round-trip failed", err)
	}
	if _, err := store.decodeConnection(id, storedConnection{
		Ciphertext: ciphertext, KeyID: keyID, Active: false, Version: 10,
	}); err == nil {
		t.Fatal("disabled credential could still be read")
	}
}
