package policies_test

import (
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/policies"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

func TestValidateUnlinkRejectsCrossOwnerOrInactive(t *testing.T) {
	link, err := factories.NewVerifiedLink("line-owner", "google-sub", []string{"calendar.read"})
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := valueobjects.NewLineUserID("line-owner")
	other, _ := valueobjects.NewLineUserID("other-line")
	subject, _ := valueobjects.NewGoogleAccountID("google-sub")
	if err := policies.ValidateUnlink(link, owner, subject); err != nil {
		t.Fatal("valid unlink denied", err)
	}
	if err := policies.ValidateUnlink(link, other, subject); err == nil {
		t.Fatal("cross-owner unlink accepted")
	}
	if err := link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if err := policies.ValidateUnlink(link, owner, subject); err == nil {
		t.Fatal("already disconnected link can be unlinked")
	}
}
