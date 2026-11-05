//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "cloud.google.com/go/firestore"
	fsadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/usecases/identity"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	oauth "github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
)

func TestOAuthSessionsAndConnectionsWithFirestoreEmulator(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Firestore emulator not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	client, err := sdk.NewClient(ctx, "linebot-oauth-emulator-ci")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	keys, err := oauth.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{0x6F}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	store, err := fsadapter.NewOAuthStore(client, keys)
	if err != nil {
		t.Fatal(err)
	}
	state := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xB7}, 32))
	session := identity.OAuthSession{
		OwnerID: "verified-user", Verifier: strings.Repeat("A", 43),
		Nonce: strings.Repeat("B", 43), Scopes: []string{"openid"},
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := store.CreateSession(ctx, state, session); err != nil {
		t.Fatal(err)
	}
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := store.ConsumeSession(ctx, state)
			if err == nil {
				if got.OwnerID != "verified-user" {
					t.Error("wrong owner")
				}
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("state consumed %d times", success.Load())
	}
	if _, err := store.ConsumeSession(ctx, state); err == nil {
		t.Fatal("replayed OAuth state accepted")
	}

	link, err := factories.NewVerifiedLink("verified-user", "google-subject", []string{"openid"})
	if err != nil {
		t.Fatal(err)
	}
	account := identity.OAuthAuthorization{
		Link: link,
		Credentials: identity.OAuthCredentials{
			AccessToken: "sensitive-access-token", RefreshToken: "sensitive-refresh-token",
			TokenType: "Bearer", Expiry: time.Now().Add(time.Hour),
		},
	}
	if err := store.SaveConnection(ctx, account); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetConnection(ctx, "verified-user", "google-subject")
	if err != nil || got.Version != 1 || got.Credentials.RefreshToken != "sensitive-refresh-token" {
		t.Fatalf("cannot read encrypted token: %v", err)
	}
	got.Credentials.AccessToken = "refreshed-access"
	got.Credentials.RefreshToken = ""
	if err := store.UpdateCredentials(ctx, got, 1); err != nil {
		t.Fatal(err)
	}
	latest, err := store.GetConnection(ctx, "verified-user", "google-subject")
	if err != nil || latest.Version != 2 || latest.Credentials.RefreshToken != "sensitive-refresh-token" {
		t.Fatal("refresh token was lost")
	}
	if err := store.UpdateCredentials(ctx, got, 1); err == nil {
		t.Fatal("stale CAS update accepted")
	}
	if err := store.DeactivateConnection(ctx, "verified-user", "google-subject"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetConnection(ctx, "verified-user", "google-subject"); err == nil {
		t.Fatal("deactivated account remained readable")
	}
	if err := store.UpdateCredentials(ctx, latest, 2); err == nil {
		t.Fatal("stale account writer reactivated revoked account")
	}
}
