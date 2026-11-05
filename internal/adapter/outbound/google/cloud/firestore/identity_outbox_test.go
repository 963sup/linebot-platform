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

func TestIdentityOutboxDocumentsMapCredentialFreeDomainEvents(t *testing.T) {
	link, err := factories.NewVerifiedLink("line-owner", "google-subject", []string{"openid"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 5, 1, 2, 3, 0, time.UTC)

	linked, err := identityOutboxDocuments("aggregate-id", 1, now, link.PendingEvents())
	if err != nil {
		t.Fatal(err)
	}
	if len(linked) != 1 {
		t.Fatalf("linked documents = %d, want 1", len(linked))
	}
	got := linked[0].record
	if got.EventID == "" || linked[0].id != got.EventID || got.EventType != "identity.account-linked" ||
		got.SchemaVersion != 1 || got.AggregateID != "aggregate-id" || got.AggregateVersion != 1 ||
		got.OccurredAt != now || got.State != "pending" ||
		got.CorrelationID == "" || got.IdempotencyKey != got.EventID || got.NextDispatchAt != now ||
		got.ExpiresAt != now.Add(30*24*time.Hour) {
		t.Fatalf("linked outbox record = %#v", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("line-owner")) || bytes.Contains(encoded, []byte("google-subject")) {
		t.Fatalf("outbox retained raw account identifiers: %s", encoded)
	}

	again, err := identityOutboxDocuments("aggregate-id", 1, now, link.PendingEvents())
	if err != nil || again[0].id != linked[0].id {
		t.Fatalf("event identity is not deterministic: first=%q second=%q err=%v", linked[0].id, again[0].id, err)
	}

	link.AcknowledgeEvents()
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	unlinked, err := identityOutboxDocuments("aggregate-id", 2, now.Add(time.Second), link.PendingEvents())
	if err != nil {
		t.Fatal(err)
	}
	if len(unlinked) != 1 || unlinked[0].record.EventType != "identity.account-unlinked" ||
		unlinked[0].record.AggregateVersion != 2 || unlinked[0].id == linked[0].id {
		t.Fatalf("unlinked outbox record = %#v", unlinked)
	}
}

func TestIdentityOutboxDocumentsRejectInvalidMetadata(t *testing.T) {
	if _, err := identityOutboxDocuments("", 0, time.Time{}, nil); err == nil {
		t.Fatal("invalid outbox metadata accepted")
	}
}

func TestIdentityOutboxStatePredicatesRejectStaleQuerySnapshots(t *testing.T) {
	now := time.Date(2026, 11, 5, 1, 2, 3, 0, time.UTC)
	expired := storedIdentityEvent{State: "publishing", LeaseUntil: now.Add(-time.Second)}
	if !identityLeaseExpired(expired, now) {
		t.Fatal("expired publishing lease not recognized")
	}
	for _, concurrent := range []storedIdentityEvent{
		{State: "published", LeaseUntil: now.Add(-time.Second)},
		{State: "unknown", LeaseUntil: now.Add(-time.Second)},
		{State: "publishing", LeaseUntil: now.Add(time.Second)},
	} {
		if identityLeaseExpired(concurrent, now) {
			t.Fatalf("concurrent terminal/live state treated as expired: %#v", concurrent)
		}
	}
	if !identityEventDue(storedIdentityEvent{State: "pending", NextDispatchAt: now}, now) {
		t.Fatal("due event not recognized")
	}
	if identityEventDue(storedIdentityEvent{State: "published", NextDispatchAt: now}, now) ||
		identityEventDue(storedIdentityEvent{State: "pending", NextDispatchAt: now.Add(time.Second)}, now) {
		t.Fatal("stale query snapshot could reserve terminal or future event")
	}
}

func TestIdentityDisconnectRequiresExactPersistedVersion(t *testing.T) {
	if !identityConnectionVersionMatches(7, 7) {
		t.Fatal("matching connection revision rejected")
	}
	for _, expected := range []int64{0, 6, 8} {
		if identityConnectionVersionMatches(7, expected) {
			t.Fatalf("stale/invalid expected revision %d accepted", expected)
		}
	}
}

func TestDisconnectedConnectionPreservesLegacyAggregateWireFormatWithoutCredentials(t *testing.T) {
	keys, err := oauth.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{0x42}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	link, err := factories.NewVerifiedLink("line-owner", "google-subject", []string{"openid"})
	if err != nil {
		t.Fatal(err)
	}
	link.AcknowledgeEvents()
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	store := &OAuthStore{keys: keys}
	now := time.Date(2026, 11, 5, 1, 2, 3, 0, time.UTC)
	saved, err := store.disconnectedConnection("aggregate-id", identity.OAuthAuthorization{
		Link: link,
		Credentials: identity.OAuthCredentials{
			AccessToken: "access-secret", RefreshToken: "refresh-secret",
		},
	}, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := keys.Open("link:aggregate-id", saved.KeyID, saved.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	var legacy legacyOAuthConnection
	if err := json.Unmarshal(plain, &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.OwnerID != "line-owner" || legacy.GoogleSubject != "google-subject" ||
		len(legacy.Scopes) != 1 || legacy.Scopes[0] != "openid" || legacy.Active || legacy.Version != 2 {
		t.Fatalf("disconnected aggregate = %#v", legacy)
	}
	if legacy.Credentials.AccessToken != "" || legacy.Credentials.RefreshToken != "" {
		t.Fatal("disconnected aggregate retained OAuth credentials")
	}
	if saved.Active || saved.Version != 2 || saved.ExpiresAt == nil || saved.UpdatedAt != now {
		t.Fatalf("stored disconnected metadata = %#v", saved)
	}
}
