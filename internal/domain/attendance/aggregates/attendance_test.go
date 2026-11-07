package aggregates

import (
	"errors"
	"testing"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func attendanceDayFixture(t *testing.T) AttendanceDay {
	t.Helper()
	id, _ := valueobjects.NewAttendanceDayID("day")
	participant, _ := valueobjects.NewParticipantID("participant")
	day, _ := valueobjects.NewWorkday("2026-11-07")
	value, err := NewAttendanceDay(id, participant, day)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAttendanceDayOrderingAndIdempotency(t *testing.T) {
	day := attendanceDayFixture(t)
	fix, _ := valueobjects.NewLocationFix(25, 121, 10)
	inID, _ := valueobjects.NewPunchID("in")
	outID, _ := valueobjects.NewPunchID("out")
	now := time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	if _, _, err := day.RecordClockOut(outID, now, fix); !errors.Is(err, ErrClockInRequired) {
		t.Fatalf("clock out first error = %v", err)
	}
	outcome, first, err := day.RecordClockIn(inID, now, fix)
	if err != nil || outcome != valueobjects.SubmissionRecorded || day.Revision() != 1 {
		t.Fatalf("clock in = %q, %v", outcome, err)
	}
	other, _ := valueobjects.NewLocationFix(24, 120, 20)
	outcome, duplicate, err := day.RecordClockIn(inID, now.Add(time.Hour), other)
	if err != nil || outcome != valueobjects.SubmissionDuplicate || duplicate.ID() != first.ID() || !duplicate.Location().Equal(fix) || day.Revision() != 1 {
		t.Fatalf("duplicate = %q, %+v, %v", outcome, duplicate, err)
	}
	if _, _, err := day.RecordClockOut(outID, now, fix); !errors.Is(err, ErrPunchOrdering) {
		t.Fatalf("ordering error = %v", err)
	}
	if outcome, _, err := day.RecordClockOut(outID, now.Add(time.Minute), fix); err != nil || outcome != valueobjects.SubmissionRecorded || day.Revision() != 2 {
		t.Fatalf("clock out = %q, %v", outcome, err)
	}
}
