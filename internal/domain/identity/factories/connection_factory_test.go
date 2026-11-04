package factories_test

import (
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

func TestRestorationDoesNotEmitNewLinkEvent(t *testing.T) {
	link, err := factories.RestoreLink("line-owner", "google-sub", []string{"calendar.read"}, valueobjects.ConnectionActive)
	if err != nil {
		t.Fatal(err)
	}
	if len(link.PendingEvents()) != 0 {
		t.Fatal("database rehydration emitted a new account-linked event")
	}
	newLink, err := factories.NewVerifiedLink("line-owner", "google-sub", []string{"calendar.read"})
	if err != nil {
		t.Fatal(err)
	}
	if len(newLink.PendingEvents()) != 1 {
		t.Fatal("verified link did not produce a link event")
	}
	if _, ok := newLink.PendingEvents()[0].(events.AccountLinked); !ok {
		t.Fatal("incorrect event type")
	}
	disconnected, err := factories.RestoreLink("line-owner", "google-sub", []string{"calendar.read"}, valueobjects.ConnectionDisconnected)
	if err != nil || disconnected.IsActive() || len(disconnected.PendingEvents()) != 0 {
		t.Fatal("invalid disconnected restoration", err)
	}
}
