package valueobjects_test

import (
	"reflect"
	"testing"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

func TestRequiredScopesAreCanonicalAndImmutable(t *testing.T) {
	scopes, err := valueobjects.NewRequiredScopes([]string{"scope.write", "scope.read", "scope.write"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"scope.read", "scope.write"}
	if got := scopes.Values(); !reflect.DeepEqual(got, want) {
		t.Fatalf("scopes = %v, want %v", got, want)
	}
	got := scopes.Values()
	got[0] = "tampered"
	if !reflect.DeepEqual(scopes.Values(), want) {
		t.Fatal("scope backing data leaked")
	}
	if !scopes.Contains("scope.write") || scopes.Contains("scope.missing") {
		t.Fatal("scope membership is incorrect")
	}
}

func TestValueObjectsRejectInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{"empty request ID", func() error { _, err := valueobjects.NewAutomationRequestID(""); return err }},
		{"padded request ID", func() error { _, err := valueobjects.NewAutomationRequestID(" id "); return err }},
		{"empty scopes", func() error { _, err := valueobjects.NewRequiredScopes(nil); return err }},
		{"spaced scope", func() error { _, err := valueobjects.NewRequiredScopes([]string{"scope read"}); return err }},
		{"empty approval", func() error { _, err := valueobjects.NewApproval(""); return err }},
		{"padded approval", func() error { _, err := valueobjects.NewApproval(" admin "); return err }},
		{"empty action", func() error { _, err := valueobjects.NewAction(""); return err }},
		{"padded action", func() error { _, err := valueobjects.NewAction(" calendar.create "); return err }},
		{"empty output", func() error { _, err := valueobjects.NewOutput(""); return err }},
		{"padded output", func() error { _, err := valueobjects.NewOutput(" done "); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}
