package specifications_test

import (
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/specifications"
)

func TestCanUseGoogleAPI(t *testing.T) {
	link, err := factories.NewVerifiedLink("line-owner", "google-sub", []string{"calendar.read", "tasks.write"})
	if err != nil {
		t.Fatal(err)
	}
	if !specifications.CanUseGoogleAPI(link, []string{"calendar.read"}) {
		t.Fatal("valid granted scope denied")
	}
	if !specifications.CanUseGoogleAPI(link, nil) {
		t.Fatal("active link incorrectly denied")
	}
	if specifications.CanUseGoogleAPI(link, []string{"forms.write"}) {
		t.Fatal("missing scope accepted")
	}
	if specifications.CanUseGoogleAPI(link, []string{""}) {
		t.Fatal("empty scope accepted")
	}
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if specifications.CanUseGoogleAPI(link, nil) {
		t.Fatal("disconnected link allowed access")
	}
}
