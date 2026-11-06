package entities

import (
	"errors"
	"testing"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func TestParticipantBindingIsIdempotentAndRejectsAnotherParticipant(t *testing.T) {
	employeeID, _ := valueobjects.NewEmployeeID("employee-1")
	p, err := NewParticipant(employeeID, "E001", "Employee One")
	if err != nil {
		t.Fatal(err)
	}
	first, _ := valueobjects.NewParticipantID("participant-1")
	if outcome, err := p.Bind(first); err != nil || outcome != valueobjects.EnrollmentBound {
		t.Fatal(err)
	}
	if outcome, err := p.Bind(first); err != nil || outcome != valueobjects.EnrollmentAlreadyBound {
		t.Fatalf("idempotent bind: %s %v", outcome, err)
	}
	other, _ := valueobjects.NewParticipantID("participant-2")
	if _, err := p.Bind(other); !errors.Is(err, ErrEnrollmentConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if err := p.Deactivate(time.Now()); err != nil || p.Active() {
		t.Fatalf("deactivate: %v", err)
	}
}

func TestEnrollmentInvitationIsSeparateFromParticipant(t *testing.T) {
	employeeID, _ := valueobjects.NewEmployeeID("employee-1")
	hash, _ := valueobjects.NewEnrollmentCodeHash("sha256-code")
	expires := time.Now().Add(time.Hour)
	invitation, err := NewEnrollmentInvitation(employeeID, hash, expires)
	if err != nil {
		t.Fatal(err)
	}
	if invitation.EmployeeID() != employeeID || invitation.CodeHash() != hash || !invitation.ExpiresAt().Equal(expires) {
		t.Fatal("invitation fields changed")
	}
}
