//go:build integration

package integration_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	fsadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func attendanceEmulatorStore(t *testing.T) (*fsadapter.AttendanceStore, time.Time) {
	t.Helper()
	client, _ := emulatorStore(t)
	store, err := fsadapter.NewAttendanceStore(client)
	if err != nil {
		t.Fatal(err)
	}
	return store, time.Now().UTC().Truncate(time.Millisecond)
}

func TestAttendanceEnrollmentAndWorkdayCASWithFirestoreEmulator(t *testing.T) {
	store, now := attendanceEmulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	employeeID, _ := valueobjects.NewEmployeeID("employee-1")
	participant, _ := entities.NewParticipant(employeeID, "E001", "Test Employee")
	codeHash, _ := valueobjects.NewEnrollmentCodeHash("enrollment-hash")
	invitation, _ := entities.NewEnrollmentInvitation(employeeID, codeHash, now.Add(time.Hour))
	if err := store.Invite(ctx, participant, invitation); err != nil {
		t.Fatal(err)
	}
	participantID, _ := valueobjects.NewParticipantID("participant-opaque")
	var bound atomic.Int32
	var replayed atomic.Int32
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
				replayed.Add(1)
			default:
				t.Errorf("unexpected enrollment outcome %q", outcome)
			}
		}()
	}
	wg.Wait()
	if bound.Load() != 1 || replayed.Load() != 7 {
		t.Fatalf("enrollment outcomes bound=%d replayed=%d", bound.Load(), replayed.Load())
	}
	resolved, err := store.FindByParticipantID(ctx, participantID)
	if err != nil || resolved.EmployeeID() != employeeID {
		t.Fatalf("resolved participant=%#v err=%v", resolved, err)
	}

	workday, _ := valueobjects.NewWorkday("2026-11-07")
	dayID, _ := valueobjects.NewAttendanceDayID("attendance-day-1")
	day, _ := aggregates.NewAttendanceDay(dayID, participantID, workday)
	fix, _ := valueobjects.NewLocationFix(25.0331, 121.5651, 8)
	clockInID, _ := valueobjects.NewPunchID("clock-in-1")
	_, _, _ = day.RecordClockIn(clockInID, now, fix)
	if err := store.Save(ctx, day, 0); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Find(ctx, participantID, workday)
	if err != nil || restored.Revision() != 1 {
		t.Fatalf("restored revision=%d err=%v", restored.Revision(), err)
	}
	punch, ok := restored.ClockIn()
	if !ok || !punch.Location().Equal(fix) {
		t.Fatalf("restored clock-in=%#v", punch)
	}

	base := restored
	var successes atomic.Int32
	var conflicts atomic.Int32
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			candidate := base
			clockOutID, _ := valueobjects.NewPunchID("clock-out-1")
			_, _, _ = candidate.RecordClockOut(clockOutID, now.Add(8*time.Hour), fix)
			err := store.Save(ctx, candidate, 1)
			if err == nil {
				successes.Add(1)
			} else if errors.Is(err, repositories.ErrConcurrentUpdate) || errors.Is(err, repositories.ErrConflict) {
				conflicts.Add(1)
			} else {
				t.Errorf("save clock-out: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || conflicts.Load() != 7 {
		t.Fatalf("CAS outcomes success=%d conflicts=%d", successes.Load(), conflicts.Load())
	}
	completed, err := store.Find(ctx, participantID, workday)
	if err != nil || completed.Revision() != 2 {
		t.Fatalf("completed revision=%d err=%v", completed.Revision(), err)
	}
}
