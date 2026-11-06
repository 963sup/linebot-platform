package aggregates

import (
	"errors"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func TestPunchPairIdempotencyWindowAndOrdering(t *testing.T) {
	a, base, location := fixture(t)
	inPunch, _ := valueobjects.NewPunchID("p-in")
	inSubmission, _ := valueobjects.NewSubmissionID("s-in")
	if outcome, err := a.RecordClockIn(inPunch, inSubmission, base, location); err != nil || outcome != valueobjects.SubmissionRecorded {
		t.Fatalf("record: %s %v", outcome, err)
	}
	if outcome, err := a.RecordClockIn(inPunch, inSubmission, base.Add(time.Hour), location); err != nil || outcome != valueobjects.SubmissionDuplicate {
		t.Fatalf("duplicate: %s %v", outcome, err)
	}
	if outcome, err := a.RecordClockIn(inPunch, inSubmission, base.Add(11*time.Hour), location); err != nil || outcome != valueobjects.SubmissionDuplicate {
		t.Fatalf("response-lost duplicate after window: %s %v", outcome, err)
	}
	outPunch, _ := valueobjects.NewPunchID("p-out")
	outSubmission, _ := valueobjects.NewSubmissionID("s-out")
	if _, err := a.RecordClockOut(outPunch, outSubmission, base, location); !errors.Is(err, ErrPunchOrdering) {
		t.Fatalf("ordering: %v", err)
	}
	if _, err := a.RecordClockOut(outPunch, outSubmission, base.Add(8*time.Hour), location); err != nil {
		t.Fatal(err)
	}
	other, _ := valueobjects.NewSubmissionID("other")
	if outcome, err := a.RecordClockOut(outPunch, other, base.Add(9*time.Hour), location); err != nil || outcome != valueobjects.SubmissionDuplicate {
		t.Fatalf("maximum pair duplicate: %s %v", outcome, err)
	}
}

func TestNewAttendanceRequiresActiveParticipantWorkplaceAndShift(t *testing.T) {
	a, base, _ := fixture(t)
	tests := []struct {
		name        string
		participant entities.Participant
		workplace   entities.Workplace
		shift       entities.ShiftInstance
	}{
		{name: "inactive participant", participant: deactivatedParticipant(t, a.Participant(), base), workplace: a.Workplace(), shift: a.Shift()},
		{name: "inactive workplace", participant: a.Participant(), workplace: deactivatedWorkplace(t, a.Workplace()), shift: a.Shift()},
		{name: "inactive shift", participant: a.Participant(), workplace: a.Workplace(), shift: deactivatedShift(t, a.Shift())},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewAttendance(tc.participant, tc.workplace, tc.shift, a.Assignment()); !errors.Is(err, ErrInvalidAttendance) {
				t.Fatalf("NewAttendance error=%v, want ErrInvalidAttendance", err)
			}
		})
	}
}

func deactivatedParticipant(t *testing.T, value entities.Participant, at time.Time) entities.Participant {
	t.Helper()
	if err := value.Deactivate(at); err != nil {
		t.Fatal(err)
	}
	return value
}

func deactivatedWorkplace(t *testing.T, value entities.Workplace) entities.Workplace {
	t.Helper()
	if err := value.Deactivate(); err != nil {
		t.Fatal(err)
	}
	return value
}

func deactivatedShift(t *testing.T, value entities.ShiftInstance) entities.ShiftInstance {
	t.Helper()
	if err := value.Deactivate(); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestSubmissionIDDoesNotCoupleIndependentBusinessSlots(t *testing.T) {
	a, base, location := fixture(t)
	inPunch, _ := valueobjects.NewPunchID("p-in")
	outPunch, _ := valueobjects.NewPunchID("p-out")
	submissionID, _ := valueobjects.NewSubmissionID("shared-submission")

	if outcome, err := a.RecordClockIn(inPunch, submissionID, base, location); err != nil || outcome != valueobjects.SubmissionRecorded {
		t.Fatalf("clock in: outcome=%s error=%v", outcome, err)
	}
	if outcome, err := a.RecordClockOut(outPunch, submissionID, base.Add(8*time.Hour), location); err != nil || outcome != valueobjects.SubmissionRecorded {
		t.Fatalf("clock out: outcome=%s error=%v", outcome, err)
	}
}

func TestAnomaliesReviewAndCorrections(t *testing.T) {
	a, base, _ := fixture(t)
	ref, _ := valueobjects.NewEvidenceRef("evidence/in")
	location, _ := valueobjects.NewLocationAssessment(200, 100, true, true, ref, "location-v1")
	punchID, _ := valueobjects.NewPunchID("p-in")
	submissionID, _ := valueobjects.NewSubmissionID("s-in")
	if _, err := a.RecordClockIn(punchID, submissionID, base.Add(-90*time.Minute), location); err != nil {
		t.Fatal(err)
	}
	punch, _ := a.ClockIn()
	if punch.ReviewState() != valueobjects.ReviewRequired || !punch.TimeAnomaly() {
		t.Fatalf("review state: %s", punch.ReviewState())
	}
	if err := a.Confirm(valueobjects.ClockIn, base.Add(time.Hour), "reviewer", "verified"); err != nil {
		t.Fatal(err)
	}
	punch, _ = a.ClockIn()
	history := punch.ReviewHistory()
	if len(history) != 1 {
		t.Fatalf("history: %d", len(history))
	}
	history[0] = entities.ReviewAction{}
	if current, _ := a.ClockIn(); current.ReviewHistory()[0].Reviewer() == "" {
		t.Fatal("mutable review history")
	}
}

func TestCorrectionPreservesPairOrdering(t *testing.T) {
	a, base, location := fixture(t)
	inPunch, _ := valueobjects.NewPunchID("p-in")
	inSubmission, _ := valueobjects.NewSubmissionID("s-in")
	outPunch, _ := valueobjects.NewPunchID("p-out")
	outSubmission, _ := valueobjects.NewSubmissionID("s-out")
	_, _ = a.RecordClockIn(inPunch, inSubmission, base, location)
	_, _ = a.RecordClockOut(outPunch, outSubmission, base.Add(8*time.Hour), location)
	if err := a.Correct(valueobjects.ClockIn, base.Add(9*time.Hour), location, valueobjects.ReviewProvenanceManualOverride, base.Add(10*time.Hour), "reviewer", "correction"); !errors.Is(err, ErrPunchOrdering) {
		t.Fatalf("correction: %v", err)
	}
	if err := a.Void(valueobjects.ClockIn, base.Add(10*time.Hour), "reviewer", "void"); !errors.Is(err, ErrPunchOrdering) {
		t.Fatalf("void: %v", err)
	}
}

func TestCorrectionRecomputesTimeAnomalyFromShiftPolicy(t *testing.T) {
	for _, tc := range []struct {
		name        string
		effectiveAt func(time.Time) time.Time
		wantAnomaly bool
	}{
		{name: "inside window", effectiveAt: func(base time.Time) time.Time { return base }, wantAnomaly: false},
		{name: "outside window", effectiveAt: func(base time.Time) time.Time { return base.Add(-time.Hour) }, wantAnomaly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, base, location := fixture(t)
			punchID, _ := valueobjects.NewPunchID("p-in")
			submissionID, _ := valueobjects.NewSubmissionID("s-in")
			if _, err := a.RecordClockIn(punchID, submissionID, base.Add(-90*time.Minute), location); err != nil {
				t.Fatal(err)
			}
			if err := a.Correct(valueobjects.ClockIn, tc.effectiveAt(base), location, valueobjects.ReviewProvenanceManualOverride, base.Add(time.Hour), "reviewer", "correct time"); err != nil {
				t.Fatal(err)
			}
			punch, _ := a.ClockIn()
			if punch.TimeAnomaly() != tc.wantAnomaly {
				t.Fatalf("time anomaly=%v, want %v", punch.TimeAnomaly(), tc.wantAnomaly)
			}
		})
	}
}

func fixture(t *testing.T) (Attendance, time.Time, valueobjects.LocationAssessment) {
	t.Helper()
	employeeID, _ := valueobjects.NewEmployeeID("employee-1")
	participantID, _ := valueobjects.NewParticipantID("participant-1")
	participant, _ := entities.NewParticipant(employeeID, "E001", "Employee One")
	_, _ = participant.Bind(participantID)
	workplaceID, _ := valueobjects.NewWorkplaceID("workplace-1")
	center, _ := valueobjects.NewGeoPoint(25, 121)
	workplace, _ := entities.DefineWorkplace(workplaceID, "Workplace One", center, 100, 50, "location-v1")
	shiftID, _ := valueobjects.NewShiftInstanceID("shift-1")
	base := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	window, _ := valueobjects.NewTimeWindow(base, base.Add(8*time.Hour))
	shift, _ := entities.DefineShiftInstance(shiftID, workplaceID, "Shift One", window, window, base.Add(-2*time.Hour), base.Add(10*time.Hour), "shift-v1")
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-1")
	assignment, _ := entities.NewShiftAssignment(assignmentID, employeeID, shiftID)
	a, err := NewAttendance(participant, workplace, shift, assignment)
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := valueobjects.NewEvidenceRef("evidence/normal")
	assessment, _ := valueobjects.NewLocationAssessment(5, 5, false, false, ref, "location-v1")
	return a, base, assessment
}
