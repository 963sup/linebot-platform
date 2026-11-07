package firestore

import (
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func TestAttendanceDocumentIDSeparatesComponents(t *testing.T) {
	a := attendanceDocumentID("ab", "c")
	b := attendanceDocumentID("a", "bc")
	if a == b || len(a) != 64 || len(b) != 64 || a != attendanceDocumentID("ab", "c") {
		t.Fatalf("invalid deterministic document IDs: %q %q", a, b)
	}
}

func TestAttendanceDayCodecRoundTripIncludesApprovedLocationFix(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	id, _ := valueobjects.NewAttendanceDayID("day-1")
	participantID, _ := valueobjects.NewParticipantID("participant-1")
	workday, _ := valueobjects.NewWorkday("2026-11-05")
	value, _ := aggregates.NewAttendanceDay(id, participantID, workday)
	fix, _ := valueobjects.NewLocationFix(25.0331, 121.5651, 8)
	clockInID, _ := valueobjects.NewPunchID("clock-in-1")
	if _, _, err := value.RecordClockIn(clockInID, now, fix); err != nil {
		t.Fatal(err)
	}
	doc := storeAttendanceDay(value, now, now)
	restored, err := restoreAttendanceDay(doc)
	if err != nil {
		t.Fatal(err)
	}
	punch, ok := restored.ClockIn()
	if !ok || punch.ID() != clockInID || !punch.Location().Equal(fix) || restored.Revision() != 1 {
		t.Fatalf("restored day=%#v punch=%#v", restored, punch)
	}
}

func TestAttendanceDaySuccessorOnlyAppendsClockOut(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	id, _ := valueobjects.NewAttendanceDayID("day-1")
	participantID, _ := valueobjects.NewParticipantID("participant-1")
	workday, _ := valueobjects.NewWorkday("2026-11-05")
	current, _ := aggregates.NewAttendanceDay(id, participantID, workday)
	fix, _ := valueobjects.NewLocationFix(25.0331, 121.5651, 8)
	clockInID, _ := valueobjects.NewPunchID("clock-in-1")
	_, _, _ = current.RecordClockIn(clockInID, now, fix)
	next := current
	clockOutID, _ := valueobjects.NewPunchID("clock-out-1")
	_, _, _ = next.RecordClockOut(clockOutID, now.Add(8*time.Hour), fix)
	if !validAttendanceDaySuccessor(current, next) {
		t.Fatal("valid clock-out successor rejected")
	}
	otherFix, _ := valueobjects.NewLocationFix(24, 120, 20)
	rewrittenIn, _ := entities.RestorePunch(clockInID, valueobjects.ClockIn, now, otherFix)
	rewritten, err := aggregates.RestoreAttendanceDay(id, participantID, workday, &rewrittenIn, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if validAttendanceDaySuccessor(rewritten, next) {
		t.Fatal("rewritten clock-in accepted")
	}
}

func TestInvitationExpiryValidationOccursBeforeIO(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	employeeID, _ := valueobjects.NewEmployeeID("employee-invite")
	participant, _ := entities.NewParticipant(employeeID, "E-INVITE", "Invite Employee")
	codeHash, _ := valueobjects.NewEnrollmentCodeHash("invite-hash")
	invitation, _ := entities.NewEnrollmentInvitation(employeeID, codeHash, now.Add(24*time.Hour+time.Nanosecond))
	store := &AttendanceStore{now: func() time.Time { return now }}
	if err := store.Invite(t.Context(), participant, invitation); err == nil {
		t.Fatal("accepted invitation expiry beyond 24 hours")
	}
}
