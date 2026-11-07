package attendance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type verifierFake struct {
	principal VerifiedPrincipal
	err       error
}

func (f verifierFake) Verify(context.Context, string) (VerifiedPrincipal, error) {
	return f.principal, f.err
}

type participantFake struct {
	value entities.Participant
	err   error
}

func (f *participantFake) FindByParticipantID(context.Context, valueobjects.ParticipantID) (entities.Participant, error) {
	return f.value, f.err
}

type enrollmentFake struct {
	value   entities.Participant
	outcome valueobjects.EnrollmentOutcome
	err     error
}

func (f *enrollmentFake) Invite(context.Context, entities.Participant, entities.EnrollmentInvitation) error {
	return f.err
}
func (f *enrollmentFake) ConsumeEnrollment(context.Context, valueobjects.EnrollmentCodeHash, valueobjects.ParticipantID, time.Time) (entities.Participant, valueobjects.EnrollmentOutcome, error) {
	return f.value, f.outcome, f.err
}

type attendanceFake struct {
	value         aggregates.AttendanceDay
	findErr       error
	saveErr       error
	commitOnError bool
	expected      uint64
}

func (f *attendanceFake) Find(context.Context, valueobjects.ParticipantID, valueobjects.Workday) (aggregates.AttendanceDay, error) {
	return f.value, f.findErr
}
func (f *attendanceFake) Save(_ context.Context, value aggregates.AttendanceDay, expected uint64) error {
	f.expected = expected
	if f.saveErr == nil || f.commitOnError {
		f.value, f.findErr = value, nil
	}
	return f.saveErr
}

func serviceFixture(t *testing.T, now time.Time) (Dependencies, valueobjects.ParticipantID) {
	t.Helper()
	participantID, _ := valueobjects.NewParticipantID("participant")
	employeeID, _ := valueobjects.NewEmployeeID("employee")
	participant, _ := entities.NewParticipant(employeeID, "E1", "Employee")
	_, _ = participant.Bind(participantID)
	return Dependencies{
		Verifier:     verifierFake{principal: VerifiedPrincipal{ParticipantID: participantID, ValidatedAt: now}},
		Participants: &participantFake{value: participant},
		Enrollment:   &enrollmentFake{value: participant, outcome: valueobjects.EnrollmentBound},
		Attendance:   &attendanceFake{findErr: repositories.ErrNotFound},
		Clock:        fixedClock{now: now},
	}, participantID
}

func TestServiceUsesTaipeiWorkdayAndServerTime(t *testing.T) {
	now := time.Date(2026, 11, 6, 16, 30, 0, 0, time.UTC)
	deps, _ := serviceFixture(t, now)
	service, err := NewService(deps)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", Kind: ClockIn, Location: LocationFix{25, 121, 9}})
	if err != nil || result.Workday != "2026-11-07" || result.Punch.RecordedAt != now || result.Status != PunchRecorded {
		t.Fatalf("Punch() = %+v, %v", result, err)
	}
	if deps.Attendance.(*attendanceFake).expected != 0 {
		t.Fatal("new day did not use revision zero")
	}
	today, err := service.GetToday(context.Background(), TodayQuery{IDToken: "token"})
	if err != nil || today.ClockIn == nil || len(today.AllowedKinds) != 1 || today.AllowedKinds[0] != ClockOut {
		t.Fatalf("GetToday() = %+v, %v", today, err)
	}
}

func TestServiceResolvesConcurrentDuplicate(t *testing.T) {
	now := time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	deps, _ := serviceFixture(t, now)
	repo := deps.Attendance.(*attendanceFake)
	repo.saveErr, repo.commitOnError = repositories.ErrConcurrentUpdate, true
	service, _ := NewService(deps)
	result, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", Kind: ClockIn, Location: LocationFix{25, 121, 9}})
	if err != nil || result.Status != PunchAlreadyRecorded {
		t.Fatalf("Punch() = %+v, %v", result, err)
	}
}

func TestServiceRequiresEnrollmentAndValidGPS(t *testing.T) {
	now := time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	deps, _ := serviceFixture(t, now)
	deps.Participants.(*participantFake).err = repositories.ErrNotFound
	service, _ := NewService(deps)
	if _, err := service.GetToday(context.Background(), TodayQuery{IDToken: "token"}); !errors.Is(err, ErrNotEnrolled) {
		t.Fatalf("GetToday error = %v", err)
	}
	deps.Participants.(*participantFake).err = nil
	if _, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", Kind: ClockIn, Location: LocationFix{91, 121, 9}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Punch error = %v", err)
	}
}

func TestEnrollPreservesBinding(t *testing.T) {
	now := time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	deps, _ := serviceFixture(t, now)
	service, _ := NewService(deps)
	result, err := service.Enroll(context.Background(), EnrollCommand{IDToken: "token", Code: "private-code"})
	if err != nil || result.Status != EnrollCreated {
		t.Fatalf("Enroll() = %+v, %v", result, err)
	}
}
