package valueobjects_test

import (
	"testing"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

func TestConnectionIDIsCompositeAndValidated(t *testing.T) {
	owner, err := valueobjects.NewLineUserID("line-owner")
	if err != nil {
		t.Fatal(err)
	}
	subject, err := valueobjects.NewGoogleAccountID("google-sub")
	if err != nil {
		t.Fatal(err)
	}
	id, err := valueobjects.NewConnectionID(owner, subject)
	if err != nil || !id.Valid() {
		t.Fatal("valid connection ID rejected", err)
	}
	if id.Owner() != owner || id.GoogleAccount() != subject {
		t.Fatal("connection ID lost identity")
	}
	if _, err := valueobjects.NewConnectionID(valueobjects.LineUserID{}, subject); err == nil {
		t.Fatal("zero owner accepted")
	}
}
