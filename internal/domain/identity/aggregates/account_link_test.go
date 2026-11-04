package aggregates_test

import (
	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/identity/events"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
	"testing"
)

func TestAccountLinkLifecycle(t *testing.T) {
	link, err := factories.NewVerifiedLink("U123", "google-subject", []string{"scope-b", "scope-a", "scope-a"})
	if err != nil {
		t.Fatal(err)
	}
	if !link.IsActive() || len(link.Scopes()) != 2 {
		t.Fatalf("unexpected new link: %#v", link)
	}
	copyScopes := link.Scopes()
	copyScopes[0] = "tampered"
	if link.Scopes()[0] == "tampered" {
		t.Fatal("scope backing data leaked")
	}
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if err := link.Disconnect(); err != nil {
		t.Fatal("disconnect must be idempotent:", err)
	}
	if link.IsActive() {
		t.Fatal("disconnected link must be inactive")
	}
}
func TestRestoreRejectsUnknownStatus(t *testing.T) {
	owner, _ := valueobjects.NewLineUserID("U123")
	subject, _ := valueobjects.NewGoogleAccountID("google-subject")
	scopes, _ := valueobjects.NewOAuthScopes([]string{"scope"})
	if _, err := aggregates.RestoreAccountLink(owner, subject, scopes, "unknown"); err == nil {
		t.Fatal("invalid status accepted")
	}
}
func TestFactoryRejectsInvalidClaims(t *testing.T) {
	cases := []struct {
		owner, subject string
		scopes         []string
	}{
		{"", "google-subject", []string{"scope"}},
		{" U123 ", "google-subject", []string{"scope"}},
		{"U123", "", []string{"scope"}},
		{"U123", "google-subject", nil},
		{"U123", "google-subject", []string{"invalid scope"}},
	}
	for _, tc := range cases {
		if _, err := factories.NewVerifiedLink(tc.owner, tc.subject, tc.scopes); err == nil {
			t.Fatalf("invalid input accepted: %+v", tc)
		}
	}
}

func TestAccountLinkEventsAreEmittedOnce(t *testing.T) {
	link, err := factories.NewVerifiedLink("line-owner", "google-subject", []string{"calendar.read"})
	if err != nil {
		t.Fatal(err)
	}
	initial := link.PendingEvents()
	if len(initial) != 1 {
		t.Fatalf("expected one link event, got %d", len(initial))
	}
	if _, ok := initial[0].(events.AccountLinked); !ok {
		t.Fatal("first event must be AccountLinked")
	}
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if len(link.PendingEvents()) != 2 {
		t.Fatal("repeated disconnect emitted duplicate events")
	}
	if _, ok := link.PendingEvents()[1].(events.AccountUnlinked); !ok {
		t.Fatal("second event must be AccountUnlinked")
	}
	id, err := link.ConnectionID()
	if err != nil || id.Owner().String() != "line-owner" {
		t.Fatal("invalid connection ID", err)
	}
}
