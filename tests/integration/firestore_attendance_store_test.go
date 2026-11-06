//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "cloud.google.com/go/firestore"
	fsadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"github.com/963sup/linebot-platform/internal/infrastructure/attendancecrypto"
)

func attendanceEmulatorStore(t *testing.T) (*fsadapter.AttendanceStore, time.Time) {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Fatal("FIRESTORE_EMULATOR_HOST is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := sdk.NewClient(ctx, fmt.Sprintf("attendance-ci-%x", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	keyring, err := attendancecrypto.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{0x51}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	store, err := fsadapter.NewAttendanceStore(client, keyring)
	if err != nil {
		t.Fatal(err)
	}
	return store, time.Now().UTC().Truncate(time.Millisecond)
}

func TestWorkplaceUpdatePreservesCreatedAtWithFirestoreEmulator(t *testing.T) {
	client, _ := emulatorStore(t)
	keyring, err := attendancecrypto.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{0x51}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	store, err := fsadapter.NewAttendanceStore(client, keyring)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	id, _ := valueobjects.NewWorkplaceID("workplace-update")
	center, _ := valueobjects.NewGeoPoint(25.033, 121.565)
	before, _ := entities.DefineWorkplace(id, "Original", center, 100, 50, "v1")
	if err := store.SaveWorkplace(ctx, before); err != nil {
		t.Fatal(err)
	}
	docs, err := client.Collection("attendance_workplaces_v1").Documents(ctx).GetAll()
	if err != nil || len(docs) != 1 {
		t.Fatalf("workplace documents=%d err=%v", len(docs), err)
	}
	// Use a distinct historical value so an accidental overwrite cannot pass
	// because two writes happened within the same clock tick.
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := docs[0].Ref.Update(ctx, []sdk.Update{{Path: "created_at", Value: created}}); err != nil {
		t.Fatal(err)
	}
	after, _ := entities.DefineWorkplace(id, "Updated", center, 200, 75, "v2")
	if err := store.SaveWorkplace(ctx, after); err != nil {
		t.Fatal(err)
	}
	snapshot, err := docs[0].Ref.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	gotCreated, err := snapshot.DataAt("created_at")
	if err != nil || gotCreated != created {
		t.Fatalf("created_at=%v err=%v; want %v", gotCreated, err, created)
	}
	restored, err := store.FindWorkplace(ctx, id)
	if err != nil || restored.DisplayName() != "Updated" || restored.PolicyVersion() != "v2" {
		t.Fatalf("updated workplace=%#v err=%v", restored, err)
	}
}

func TestAttendanceEnrollmentPunchAndEncryptedEvidenceWithFirestoreEmulator(t *testing.T) {
	store, now := attendanceEmulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	employeeID, _ := valueobjects.NewEmployeeID("employee-1")
	participant, err := entities.NewParticipant(employeeID, "E001", "Test Employee")
	if err != nil {
		t.Fatal(err)
	}
	codeHash, _ := valueobjects.NewEnrollmentCodeHash("v1:enrollment-hash")
	invitation, _ := entities.NewEnrollmentInvitation(employeeID, codeHash, now.Add(time.Hour))
	if err := store.Invite(ctx, participant, invitation); err != nil {
		t.Fatal(err)
	}
	participantID, _ := valueobjects.NewParticipantID("participant:v1:opaque")
	var bound atomic.Int32
	var alreadyBound atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, outcome, err := store.ConsumeEnrollment(ctx, codeHash, participantID, now)
			if err != nil {
				t.Errorf("consume enrollment: %v", err)
				return
			}
			switch outcome {
			case valueobjects.EnrollmentBound:
				bound.Add(1)
			case valueobjects.EnrollmentAlreadyBound:
				alreadyBound.Add(1)
			default:
				t.Errorf("unexpected outcome %q", outcome)
			}
		}()
	}
	wg.Wait()
	if bound.Load() != 1 || alreadyBound.Load() != 7 {
		t.Fatalf("enrollment outcomes bound=%d already=%d", bound.Load(), alreadyBound.Load())
	}
	expiringEmployeeID, _ := valueobjects.NewEmployeeID("employee-expiring")
	expiringParticipant, _ := entities.NewParticipant(expiringEmployeeID, "E-EXP", "Expiring Employee")
	expiringHash, _ := valueobjects.NewEnrollmentCodeHash("v1:expiring-enrollment-hash")
	expiresAt := time.Now().UTC().Add(time.Hour)
	expiringInvitation, _ := entities.NewEnrollmentInvitation(expiringEmployeeID, expiringHash, expiresAt)
	if err := store.Invite(ctx, expiringParticipant, expiringInvitation); err != nil {
		t.Fatal(err)
	}
	expiringParticipantID, _ := valueobjects.NewParticipantID("participant-expiring")
	if _, _, err := store.ConsumeEnrollment(ctx, expiringHash, expiringParticipantID, expiresAt); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("enrollment accepted exactly at expiry: %v", err)
	}

	workplaceID, _ := valueobjects.NewWorkplaceID("workplace-1")
	center, _ := valueobjects.NewGeoPoint(25.033, 121.565)
	workplace, _ := entities.DefineWorkplace(workplaceID, "Taipei Office", center, 100, 50, "workplace-v1")
	if err := store.SaveWorkplace(ctx, workplace); err != nil {
		t.Fatal(err)
	}
	shiftID, _ := valueobjects.NewShiftInstanceID("shift-1")
	clockIn, _ := valueobjects.NewTimeWindow(now.Add(-time.Hour), now.Add(time.Hour))
	clockOut, _ := valueobjects.NewTimeWindow(now.Add(7*time.Hour), now.Add(9*time.Hour))
	shift, _ := entities.DefineShiftInstance(shiftID, workplaceID, "Day Shift", clockIn, clockOut, now.Add(-2*time.Hour), now.Add(10*time.Hour), "shift-v1")
	if err := store.SaveShift(ctx, shift); err != nil {
		t.Fatal(err)
	}
	changedShift, _ := entities.DefineShiftInstance(shiftID, workplaceID, "Changed Day Shift", clockIn, clockOut, now.Add(-2*time.Hour), now.Add(10*time.Hour), "shift-v1")
	if err := store.SaveShift(ctx, changedShift); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("mutable shift definition error=%v", err)
	}
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-1")
	assignment, _ := entities.NewShiftAssignment(assignmentID, employeeID, shiftID)
	if err := store.SaveAssignment(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	options, err := store.ListCurrentOptions(ctx, participantID, now)
	if err != nil || len(options) != 1 || options[0].Assignment.ID() != assignmentID || options[0].Attendance != nil {
		t.Fatalf("current options before punch=%#v err=%v", options, err)
	}
	punchContext, err := store.ResolvePunchContext(ctx, participantID, shiftID, now)
	if err != nil || punchContext.Assignment.ID() != assignmentID || punchContext.Attendance != nil {
		t.Fatalf("punch context before punch=%#v err=%v", punchContext, err)
	}
	otherEmployeeID, _ := valueobjects.NewEmployeeID("employee-other")
	conflictingAssignment, _ := entities.NewShiftAssignment(assignmentID, otherEmployeeID, shiftID)
	if err := store.SaveAssignment(ctx, conflictingAssignment); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("assignment identity rebind error=%v", err)
	}
	duplicateAssignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-duplicate")
	duplicateAssignment, _ := entities.NewShiftAssignment(duplicateAssignmentID, employeeID, shiftID)
	if err := store.SaveAssignment(ctx, duplicateAssignment); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("duplicate employee/shift assignment error=%v", err)
	}
	historicalBase := now.Add(-48 * time.Hour)
	historicalShiftID, _ := valueobjects.NewShiftInstanceID("shift-historical")
	historicalClockIn, _ := valueobjects.NewTimeWindow(historicalBase.Add(-time.Hour), historicalBase.Add(time.Hour))
	historicalClockOut, _ := valueobjects.NewTimeWindow(historicalBase.Add(7*time.Hour), historicalBase.Add(9*time.Hour))
	historicalShift, _ := entities.DefineShiftInstance(historicalShiftID, workplaceID, "Historical Shift", historicalClockIn, historicalClockOut, historicalBase.Add(-2*time.Hour), historicalBase.Add(10*time.Hour), "shift-v1")
	if err := store.SaveShift(ctx, historicalShift); err != nil {
		t.Fatal(err)
	}
	historicalAssignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-historical")
	historicalAssignment, _ := entities.NewShiftAssignment(historicalAssignmentID, employeeID, historicalShiftID)
	if err := store.SaveAssignment(ctx, historicalAssignment); err != nil {
		t.Fatal(err)
	}
	options, err = store.ListCurrentOptions(ctx, participantID, now)
	if err != nil || len(options) != 1 || options[0].Assignment.ID() != assignmentID {
		t.Fatalf("current options include historical assignment: %#v err=%v", options, err)
	}
	boundParticipant, err := store.FindByParticipantID(ctx, participantID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := aggregates.NewAttendance(boundParticipant, workplace, shift, assignment)
	if err != nil {
		t.Fatal(err)
	}
	punchID, _ := valueobjects.NewPunchID("punch-1")
	evidenceRef, _ := valueobjects.NewEvidenceRef(string(punchID))
	evidence, _ := valueobjects.NewLocationEvidence(25.0331, 121.5651, 8)
	assessment, _ := workplace.AssessLocation(evidence, evidenceRef)
	submissionID, _ := valueobjects.NewSubmissionID("submission-1")
	if _, err := record.RecordClockIn(punchID, submissionID, now, assessment); err != nil {
		t.Fatal(err)
	}
	if err := store.SavePunch(ctx, record, 0, punchID, evidence, now.Add(90*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	punchContext, err = store.ResolvePunchContext(ctx, participantID, shiftID, now)
	if err != nil || punchContext.Attendance == nil || punchContext.Attendance.Revision() != 1 {
		t.Fatalf("punch context after punch=%#v err=%v", punchContext, err)
	}
	restored, err := store.Find(ctx, participantID, assignmentID)
	if err != nil || restored.Revision() != 1 {
		t.Fatalf("restored revision=%d err=%v", restored.Revision(), err)
	}
	decrypted, err := store.ReadEvidence(ctx, evidenceRef, "reviewer@example.test", "investigate location anomaly", now.Add(time.Minute))
	if err != nil || decrypted.Latitude() != evidence.Latitude() || decrypted.Longitude() != evidence.Longitude() {
		t.Fatalf("decrypted=%#v err=%v", decrypted, err)
	}
	if _, err := store.ReadEvidence(ctx, evidenceRef, "", "", now); err == nil {
		t.Fatal("raw evidence access without reviewer and reason succeeded")
	}
	correctedLocation, _ := valueobjects.NewLocationAssessment(3, evidence.AccuracyMeters(), false, false, evidenceRef, workplace.PolicyVersion())
	if err := record.Correct(valueobjects.ClockIn, now.Add(2*time.Minute), correctedLocation, valueobjects.ReviewProvenanceManualOverride, now.Add(3*time.Minute), "operator-claim", "correct recorded time"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReview(ctx, record, 1); err != nil {
		t.Fatal(err)
	}
	reviewed, err := store.Find(ctx, participantID, assignmentID)
	if err != nil {
		t.Fatal(err)
	}
	reviewedPunch, ok := reviewed.ClockIn()
	if !ok || len(reviewedPunch.ReviewHistory()) != 1 {
		t.Fatalf("reviewed punch=%#v", reviewedPunch)
	}
	action := reviewedPunch.ReviewHistory()[0]
	if action.Provenance() != valueobjects.ReviewProvenanceManualOverride ||
		!action.Before().EffectiveAt().Equal(now) || !action.After().EffectiveAt().Equal(now.Add(2*time.Minute)) ||
		action.Before().Location().EvidenceRef() != action.After().Location().EvidenceRef() {
		t.Fatalf("review action=%#v", action)
	}
	if err := boundParticipant.Deactivate(now.Add(2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveParticipant(ctx, boundParticipant); err != nil {
		t.Fatal(err)
	}
	inactive, err := store.FindByParticipantID(ctx, participantID)
	if err != nil || inactive.Active() || inactive.DeactivatedAt().IsZero() {
		t.Fatalf("deactivated participant=%#v err=%v", inactive, err)
	}
}

func TestAttendanceConcurrentPunchCASWithFirestoreEmulator(t *testing.T) {
	store, now := attendanceEmulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// Reuse the first integration test's complete setup in each independent
	// emulator project by calling its operations through a compact fixture.
	participantID, assignmentID, record, workplace := seedAttendanceRecord(t, ctx, store, now)
	evidence, _ := valueobjects.NewLocationEvidence(25.0331, 121.5651, 8)
	var success atomic.Int32
	var conflicts atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			candidate := record
			punchID, _ := valueobjects.NewPunchID("clock-out-punch")
			ref, _ := valueobjects.NewEvidenceRef(string(punchID))
			assessment, _ := workplace.AssessLocation(evidence, ref)
			submission, _ := valueobjects.NewSubmissionID("clock-out-submission")
			if _, err := candidate.RecordClockOut(punchID, submission, now.Add(8*time.Hour), assessment); err != nil {
				t.Error(err)
				return
			}
			err := store.SavePunch(ctx, candidate, 1, punchID, evidence, now.Add(8*time.Hour+90*24*time.Hour))
			if err == nil {
				success.Add(1)
			} else if errors.Is(err, repositories.ErrConcurrentUpdate) || errors.Is(err, repositories.ErrConflict) {
				conflicts.Add(1)
			} else {
				t.Errorf("save punch: %v", err)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || conflicts.Load() != 7 {
		t.Fatalf("CAS outcomes success=%d conflicts=%d", success.Load(), conflicts.Load())
	}
	got, err := store.Find(ctx, participantID, assignmentID)
	if err != nil || got.Revision() != 2 {
		t.Fatalf("final revision=%d err=%v", got.Revision(), err)
	}
}

func TestAttendanceReviewListPaginatesPastExpiredRows(t *testing.T) {
	store, now := attendanceEmulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	seedReviewRecord(t, ctx, store, "expired", now.AddDate(-5, 0, 0).Add(-time.Hour))
	time.Sleep(5 * time.Millisecond)
	activeAssignment := seedReviewRecord(t, ctx, store, "active", now)
	items, err := store.ListReviewRequired(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Assignment().ID() != activeAssignment {
		t.Fatalf("review queue=%v, want active assignment %s", items, activeAssignment)
	}
}

func seedReviewRecord(t *testing.T, ctx context.Context, store *fsadapter.AttendanceStore, suffix string, punchTime time.Time) valueobjects.ShiftAssignmentID {
	t.Helper()
	employeeID, _ := valueobjects.NewEmployeeID("employee-review-" + suffix)
	participant, _ := entities.NewParticipant(employeeID, "E-"+suffix, "Review "+suffix)
	participantID, _ := valueobjects.NewParticipantID("participant-review-" + suffix)
	codeHash, _ := valueobjects.NewEnrollmentCodeHash("v1:review-enrollment-" + suffix)
	invitation, _ := entities.NewEnrollmentInvitation(employeeID, codeHash, time.Now().UTC().Add(time.Hour))
	if err := store.Invite(ctx, participant, invitation); err != nil {
		t.Fatal(err)
	}
	participant, _, err := store.ConsumeEnrollment(ctx, codeHash, participantID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	workplaceID, _ := valueobjects.NewWorkplaceID("workplace-review-" + suffix)
	center, _ := valueobjects.NewGeoPoint(25.033, 121.565)
	workplace, _ := entities.DefineWorkplace(workplaceID, "Review Office "+suffix, center, 100, 20, "workplace-v1")
	if err := store.SaveWorkplace(ctx, workplace); err != nil {
		t.Fatal(err)
	}
	shiftID, _ := valueobjects.NewShiftInstanceID("shift-review-" + suffix)
	clockIn, _ := valueobjects.NewTimeWindow(punchTime.Add(-time.Hour), punchTime.Add(time.Hour))
	clockOut, _ := valueobjects.NewTimeWindow(punchTime.Add(7*time.Hour), punchTime.Add(9*time.Hour))
	shift, _ := entities.DefineShiftInstance(shiftID, workplaceID, "Review Shift "+suffix, clockIn, clockOut, punchTime.Add(-2*time.Hour), punchTime.Add(10*time.Hour), "shift-v1")
	if err := store.SaveShift(ctx, shift); err != nil {
		t.Fatal(err)
	}
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-review-" + suffix)
	assignment, _ := entities.NewShiftAssignment(assignmentID, employeeID, shiftID)
	if err := store.SaveAssignment(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	record, _ := aggregates.NewAttendance(participant, workplace, shift, assignment)
	punchID, _ := valueobjects.NewPunchID("punch-review-" + suffix)
	ref, _ := valueobjects.NewEvidenceRef(string(punchID))
	evidence, _ := valueobjects.NewLocationEvidence(25.0331, 121.5651, 100)
	assessment, _ := workplace.AssessLocation(evidence, ref)
	submission, _ := valueobjects.NewSubmissionID("submission-review-" + suffix)
	_, _ = record.RecordClockIn(punchID, submission, punchTime, assessment)
	if err := store.SavePunch(ctx, record, 0, punchID, evidence, punchTime.Add(90*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	return assignmentID
}

func seedAttendanceRecord(t *testing.T, ctx context.Context, store *fsadapter.AttendanceStore, now time.Time) (valueobjects.ParticipantID, valueobjects.ShiftAssignmentID, aggregates.Attendance, entities.Workplace) {
	t.Helper()
	employeeID, _ := valueobjects.NewEmployeeID("employee-cas")
	participant, _ := entities.NewParticipant(employeeID, "E-CAS", "CAS Employee")
	participantID, _ := valueobjects.NewParticipantID("participant-cas")
	codeHash, _ := valueobjects.NewEnrollmentCodeHash("v1:cas-enrollment-hash")
	invitation, _ := entities.NewEnrollmentInvitation(employeeID, codeHash, now.Add(time.Hour))
	if err := store.Invite(ctx, participant, invitation); err != nil {
		t.Fatal(err)
	}
	participant, _, err := store.ConsumeEnrollment(ctx, codeHash, participantID, now)
	if err != nil {
		t.Fatal(err)
	}
	workplaceID, _ := valueobjects.NewWorkplaceID("workplace-cas")
	center, _ := valueobjects.NewGeoPoint(25.033, 121.565)
	workplace, _ := entities.DefineWorkplace(workplaceID, "CAS Office", center, 100, 50, "workplace-v1")
	if err := store.SaveWorkplace(ctx, workplace); err != nil {
		t.Fatal(err)
	}
	shiftID, _ := valueobjects.NewShiftInstanceID("shift-cas")
	clockIn, _ := valueobjects.NewTimeWindow(now.Add(-time.Hour), now.Add(time.Hour))
	clockOut, _ := valueobjects.NewTimeWindow(now.Add(7*time.Hour), now.Add(9*time.Hour))
	shift, _ := entities.DefineShiftInstance(shiftID, workplaceID, "CAS Shift", clockIn, clockOut, now.Add(-2*time.Hour), now.Add(10*time.Hour), "shift-v1")
	if err := store.SaveShift(ctx, shift); err != nil {
		t.Fatal(err)
	}
	assignmentID, _ := valueobjects.NewShiftAssignmentID("assignment-cas")
	assignment, _ := entities.NewShiftAssignment(assignmentID, employeeID, shiftID)
	if err := store.SaveAssignment(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	record, _ := aggregates.NewAttendance(participant, workplace, shift, assignment)
	punchID, _ := valueobjects.NewPunchID("clock-in-punch")
	ref, _ := valueobjects.NewEvidenceRef(string(punchID))
	evidence, _ := valueobjects.NewLocationEvidence(25.0331, 121.5651, 8)
	assessment, _ := workplace.AssessLocation(evidence, ref)
	submission, _ := valueobjects.NewSubmissionID("clock-in-submission")
	_, _ = record.RecordClockIn(punchID, submission, now, assessment)
	if err := store.SavePunch(ctx, record, 0, punchID, evidence, now.Add(90*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	return participantID, assignmentID, record, workplace
}
