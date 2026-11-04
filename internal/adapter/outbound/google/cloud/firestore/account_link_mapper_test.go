package firestore

import (
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

func TestAccountLinkMapperPreservesIdentityAndState(t *testing.T) {
	conn := outbound.OAuthConnection{
		OwnerID:       "line-123",
		GoogleSubject: "google-456",
		Scopes:        []string{"scope-b", "scope-a", "scope-a"},
		Credentials:   outbound.OAuthCredentials{AccessToken: "sensitive"},
		Active:        true,
	}
	link, err := accountLinkFromConnection(conn)
	if err != nil {
		t.Fatal(err)
	}
	if link.Owner().String() != conn.OwnerID || link.GoogleAccount().String() != conn.GoogleSubject || !link.IsActive() {
		t.Fatal("identity or active status lost")
	}
	if len(link.PendingEvents()) != 0 {
		t.Fatal("rehydrated account emitted a new link event")
	}
	if len(link.Scopes()) != 2 {
		t.Fatal("scopes not canonicalized")
	}
	conn.Scopes[0] = "mutated"
	if link.Scopes()[0] == "mutated" {
		t.Fatal("domain aggregate aliases persistence DTO")
	}
	conn.Active = false
	inactive, err := accountLinkFromConnection(conn)
	if err != nil || inactive.Status() != valueobjects.ConnectionDisconnected {
		t.Fatal("disconnected state not preserved", err)
	}
}

func TestAccountLinkMapperRejectsInvalidLegacyIdentity(t *testing.T) {
	for _, conn := range []outbound.OAuthConnection{
		{OwnerID: "", GoogleSubject: "sub", Scopes: []string{"scope"}, Active: true},
		{OwnerID: "line", GoogleSubject: "", Scopes: []string{"scope"}, Active: true},
		{OwnerID: "line", GoogleSubject: "sub", Scopes: nil, Active: true},
		{OwnerID: "line", GoogleSubject: "sub", Scopes: []string{"bad scope"}, Active: true},
	} {
		if _, err := accountLinkFromConnection(conn); err == nil {
			t.Fatalf("invalid persisted link accepted: %+v", conn)
		}
	}
}
