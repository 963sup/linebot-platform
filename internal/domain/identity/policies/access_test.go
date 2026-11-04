package policies_test

import (
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/policies"
	"testing"
)

func TestCanUseGoogleAPI(t *testing.T) {
	link, err := factories.NewVerifiedLink("U123", "sub", []string{"calendar.read", "tasks.write"})
	if err != nil {
		t.Fatal(err)
	}
	if !policies.CanUseGoogleAPI(link, []string{"calendar.read"}) {
		t.Fatal("valid scope denied")
	}
	if policies.CanUseGoogleAPI(link, []string{"forms.write"}) {
		t.Fatal("missing scope allowed")
	}
	if policies.CanUseGoogleAPI(link, []string{""}) {
		t.Fatal("empty scope allowed")
	}
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if policies.CanUseGoogleAPI(link, []string{"calendar.read"}) {
		t.Fatal("disconnected link allowed")
	}
}
