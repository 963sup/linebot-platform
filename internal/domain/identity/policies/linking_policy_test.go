package policies_test

import (
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/identity/policies"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

func TestValidateGrantedScopes(t *testing.T) {
	requested, _ := valueobjects.NewOAuthScopes([]string{"openid", "email", "calendar.read"})
	if err := policies.ValidateGrantedScopes(requested, []string{"calendar.read"}); err != nil {
		t.Fatal("valid consent denied", err)
	}
	if err := policies.ValidateGrantedScopes(requested, []string{"tasks.read"}); err == nil {
		t.Fatal("missing required scope accepted")
	}
	identityOnly, _ := valueobjects.NewOAuthScopes([]string{"openid", "email"})
	if err := policies.ValidateGrantedScopes(identityOnly, nil); err != nil {
		t.Fatal("verified identity-only consent changed behavior", err)
	}
}
