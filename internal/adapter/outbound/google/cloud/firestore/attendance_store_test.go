package firestore

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func TestAttendanceDocumentIDSeparatesComponents(t *testing.T) {
	a := attendanceDocumentID("ab", "c")
	b := attendanceDocumentID("a", "bc")
	if a == b || len(a) != 64 || len(b) != 64 {
		t.Fatalf("ambiguous document IDs: %q %q", a, b)
	}
	if a != attendanceDocumentID("ab", "c") {
		t.Fatal("document ID is not deterministic")
	}
}

func TestAttendanceRecordCodecExcludesRawPunchCoordinates(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	participant := testAttendanceParticipant(t)
	workplace := testAttendanceWorkplace(t)
	shift := testAttendanceShift(t, now, workplace.ID())
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-1")
	assignment, err := entities.NewShiftAssignment(assignmentID, participant.EmployeeID(), shift.ID())
	if err != nil {
		t.Fatal(err)
	}
	record, err := aggregates.NewAttendance(participant, workplace, shift, assignment)
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := valueobjects.NewEvidenceRef("punch-1")
	assessment, err := valueobjects.NewLocationAssessment(42.5, 7.25, false, false, ref, "workplace-policy-v1")
	if err != nil {
		t.Fatal(err)
	}
	punchID, _ := valueobjects.NewPunchID("punch-1")
	submissionID, _ := valueobjects.NewSubmissionID("submission-1")
	if _, err := record.RecordClockIn(punchID, submissionID, now, assessment); err != nil {
		t.Fatal(err)
	}
	doc, err := storeAttendanceRecord(record, now)
	if err != nil {
		t.Fatal(err)
	}
	doc.CreatedAt = now
	doc.ExpiresAt = attendanceExpiry(now)
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, forbidden := range []string{"11.111111", "22.222222", "ciphertext"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("aggregate document contains raw evidence %q: %s", forbidden, text)
		}
	}
	restored, err := restoreAttendanceRecord(doc)
	if err != nil {
		t.Fatal(err)
	}
	punch, ok := restored.ClockIn()
	if !ok || punch.ID() != punchID || punch.Location().EvidenceRef() != ref || restored.Revision() != 1 {
		t.Fatalf("restored attendance = %#v punch=%#v", restored, punch)
	}
}

func TestAttendanceParticipantCodecAppliesRetentionOnlyAfterDeactivation(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	participant := testAttendanceParticipant(t)
	active := storeAttendanceParticipant(participant, now, now)
	if active.ExpiresAt != nil || active.DeactivatedAt != nil {
		t.Fatalf("active participant has expiry: %#v", active)
	}
	if err := participant.Deactivate(now); err != nil {
		t.Fatal(err)
	}
	inactive := storeAttendanceParticipant(participant, now, now)
	if inactive.ExpiresAt == nil || !inactive.ExpiresAt.Equal(attendanceExpiry(now)) {
		t.Fatalf("deactivated participant expiry = %v", inactive.ExpiresAt)
	}
	if _, err := restoreAttendanceParticipant(inactive); err != nil {
		t.Fatal(err)
	}
}

func TestAttendanceInvitationRejectsExpiryBeyondTwentyFourHoursBeforeIO(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	employeeID, _ := valueobjects.NewEmployeeID("employee-invite")
	participant, _ := entities.NewParticipant(employeeID, "E-INVITE", "Invite Employee")
	codeHash, _ := valueobjects.NewEnrollmentCodeHash("v1:invite-hash")
	invitation, _ := entities.NewEnrollmentInvitation(employeeID, codeHash, now.Add(24*time.Hour+time.Nanosecond))
	store := &AttendanceStore{now: func() time.Time { return now }}
	if err := store.Invite(t.Context(), participant, invitation); err == nil {
		t.Fatal("accepted invitation expiry beyond 24 hours")
	}
}

func TestAttendanceRetentionBoundaryFailsClosed(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	if !retentionExpired(now, now) || !retentionExpired(now.Add(-time.Nanosecond), now) {
		t.Fatal("expired retention boundary remained readable")
	}
	if retentionExpired(now.Add(time.Nanosecond), now) || retentionExpired(time.Time{}, now) {
		t.Fatal("future or absent retention boundary reported expired")
	}
}

func TestAttendanceReviewUpdateRejectsRewrittenHistoryPrefix(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	participant := testAttendanceParticipant(t)
	workplace := testAttendanceWorkplace(t)
	shift := testAttendanceShift(t, now, workplace.ID())
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-review")
	assignment, _ := entities.NewShiftAssignment(assignmentID, participant.EmployeeID(), shift.ID())
	record, _ := aggregates.NewAttendance(participant, workplace, shift, assignment)
	punchID, _ := valueobjects.NewPunchID("punch-review")
	submissionID, _ := valueobjects.NewSubmissionID("submission-review")
	evidenceRef, _ := valueobjects.NewEvidenceRef(string(punchID))
	requiresReview, _ := valueobjects.NewLocationAssessment(150, 5, true, false, evidenceRef, workplace.PolicyVersion())
	if _, err := record.RecordClockIn(punchID, submissionID, now, requiresReview); err != nil {
		t.Fatal(err)
	}
	if err := record.Confirm(valueobjects.ClockIn, now.Add(time.Hour), "operator-claim", "confirm anomaly"); err != nil {
		t.Fatal(err)
	}
	current, err := storeAttendanceRecord(record, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	current.CreatedAt = now
	current.ExpiresAt = attendanceExpiry(now)
	incomingAggregate, err := restoreAttendanceRecord(current)
	if err != nil {
		t.Fatal(err)
	}
	corrected, _ := valueobjects.NewLocationAssessment(5, 5, false, false, evidenceRef, workplace.PolicyVersion())
	if err := incomingAggregate.Correct(valueobjects.ClockIn, now.Add(5*time.Minute), corrected, valueobjects.ReviewProvenanceManualOverride, now.Add(2*time.Hour), "operator-claim", "manual correction"); err != nil {
		t.Fatal(err)
	}
	incoming, err := storeAttendanceRecord(incomingAggregate, now.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAttendanceUpdate(current, incoming, false); err != nil {
		t.Fatalf("valid append rejected: %v", err)
	}
	incoming.ClockIn.History[0].Reason = "rewritten prior reason"
	if err := validateAttendanceUpdate(current, incoming, false); err == nil {
		t.Fatal("rewritten review history prefix accepted")
	}
}

func TestAttendancePunchUpdateComparesTimestampInstants(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	participant := testAttendanceParticipant(t)
	workplace := testAttendanceWorkplace(t)
	shift := testAttendanceShift(t, now, workplace.ID())
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-time-instant")
	assignment, _ := entities.NewShiftAssignment(assignmentID, participant.EmployeeID(), shift.ID())
	record, _ := aggregates.NewAttendance(participant, workplace, shift, assignment)
	rawEvidence, _ := valueobjects.NewLocationEvidence(25.0331, 121.5651, 8)
	clockInID, _ := valueobjects.NewPunchID("clock-in-time-instant")
	clockInRef, _ := valueobjects.NewEvidenceRef(string(clockInID))
	clockInLocation, _ := workplace.AssessLocation(rawEvidence, clockInRef)
	clockInSubmission, _ := valueobjects.NewSubmissionID("clock-in-time-instant")
	if _, err := record.RecordClockIn(clockInID, clockInSubmission, now, clockInLocation); err != nil {
		t.Fatal(err)
	}
	current, err := storeAttendanceRecord(record, now)
	if err != nil {
		t.Fatal(err)
	}
	current.CreatedAt = now
	current.ExpiresAt = attendanceExpiry(now)
	clockOutID, _ := valueobjects.NewPunchID("clock-out-time-instant")
	clockOutRef, _ := valueobjects.NewEvidenceRef(string(clockOutID))
	clockOutLocation, _ := workplace.AssessLocation(rawEvidence, clockOutRef)
	clockOutSubmission, _ := valueobjects.NewSubmissionID("clock-out-time-instant")
	if _, err := record.RecordClockOut(clockOutID, clockOutSubmission, now.Add(8*time.Hour), clockOutLocation); err != nil {
		t.Fatal(err)
	}
	incoming, err := storeAttendanceRecord(record, now.Add(8*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	zone := time.FixedZone("wire-offset", 8*60*60)
	current.ClockIn.RecordedAt = current.ClockIn.RecordedAt.In(zone)
	current.ClockIn.EffectiveAt = current.ClockIn.EffectiveAt.In(zone)
	current.Assignment.SubmissionOpensAt = current.Assignment.SubmissionOpensAt.In(zone)
	current.Assignment.SubmissionClosesAt = current.Assignment.SubmissionClosesAt.In(zone)
	current.Shift.SubmissionOpensAt = current.Shift.SubmissionOpensAt.In(zone)
	current.Shift.SubmissionClosesAt = current.Shift.SubmissionClosesAt.In(zone)
	if err := validateAttendanceUpdate(current, incoming, true); err != nil {
		t.Fatalf("equal timestamp instants rejected after wire round trip: %v", err)
	}
}

func TestAttendanceSavePunchRejectsMismatchedEvidenceBeforeIO(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	participant := testAttendanceParticipant(t)
	workplace := testAttendanceWorkplace(t)
	shift := testAttendanceShift(t, now, workplace.ID())
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-evidence")
	assignment, _ := entities.NewShiftAssignment(assignmentID, participant.EmployeeID(), shift.ID())
	punchID, _ := valueobjects.NewPunchID("punch-evidence")
	submissionID, _ := valueobjects.NewSubmissionID("submission-evidence")
	rawEvidence, _ := valueobjects.NewLocationEvidence(25.0331, 121.5651, 8)
	store := &AttendanceStore{now: func() time.Time { return now }}

	t.Run("reference belongs to another punch", func(t *testing.T) {
		record, _ := aggregates.NewAttendance(participant, workplace, shift, assignment)
		wrongRef, _ := valueobjects.NewEvidenceRef("another-punch")
		assessment, _ := workplace.AssessLocation(rawEvidence, wrongRef)
		if _, err := record.RecordClockIn(punchID, submissionID, now, assessment); err != nil {
			t.Fatal(err)
		}
		if err := store.SavePunch(t.Context(), record, 0, punchID, rawEvidence, now.Add(locationEvidenceRetention)); err == nil {
			t.Fatal("evidence reference owned by another punch accepted")
		}
	})

	t.Run("assessment disagrees with raw evidence", func(t *testing.T) {
		record, _ := aggregates.NewAttendance(participant, workplace, shift, assignment)
		ref, _ := valueobjects.NewEvidenceRef(string(punchID))
		assessment, _ := valueobjects.NewLocationAssessment(0, rawEvidence.AccuracyMeters(), false, false, ref, workplace.PolicyVersion())
		if _, err := record.RecordClockIn(punchID, submissionID, now, assessment); err != nil {
			t.Fatal(err)
		}
		if err := store.SavePunch(t.Context(), record, 0, punchID, rawEvidence, now.Add(locationEvidenceRetention)); err == nil {
			t.Fatal("evidence inconsistent with stored assessment accepted")
		}
	})
}

func testAttendanceParticipant(t *testing.T) entities.Participant {
	t.Helper()
	employeeID, _ := valueobjects.NewEmployeeID("employee-1")
	participant, err := entities.NewParticipant(employeeID, "E001", "Test Employee")
	if err != nil {
		t.Fatal(err)
	}
	participantID, _ := valueobjects.NewParticipantID("participant-v1")
	if _, err := participant.Bind(participantID); err != nil {
		t.Fatal(err)
	}
	return participant
}

func testAttendanceWorkplace(t *testing.T) entities.Workplace {
	t.Helper()
	id, _ := valueobjects.NewWorkplaceID("workplace-1")
	center, _ := valueobjects.NewGeoPoint(25.033, 121.565)
	workplace, err := entities.DefineWorkplace(id, "Taipei Office", center, 100, 50, "workplace-policy-v1")
	if err != nil {
		t.Fatal(err)
	}
	return workplace
}

func testAttendanceShift(t *testing.T, now time.Time, workplaceID valueobjects.WorkplaceID) entities.ShiftInstance {
	t.Helper()
	id, _ := valueobjects.NewShiftInstanceID("shift-1")
	clockIn, _ := valueobjects.NewTimeWindow(now.Add(-time.Hour), now.Add(time.Hour))
	clockOut, _ := valueobjects.NewTimeWindow(now.Add(7*time.Hour), now.Add(9*time.Hour))
	shift, err := entities.DefineShiftInstance(id, workplaceID, "Day Shift", clockIn, clockOut, now.Add(-2*time.Hour), now.Add(10*time.Hour), "shift-policy-v1")
	if err != nil {
		t.Fatal(err)
	}
	return shift
}
