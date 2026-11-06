package attendance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type tokenGeneratorFake struct {
	value string
	err   error
	used  bool
}

func (f *tokenGeneratorFake) Generate() (string, error) {
	f.used = true
	return f.value, f.err
}

type evidenceReaderFake struct {
	value      valueobjects.LocationEvidence
	reviewer   string
	reason     string
	accessedAt time.Time
}

func (f *evidenceReaderFake) ReadEvidence(_ context.Context, _ valueobjects.EvidenceRef, reviewer, reason string, accessedAt time.Time) (valueobjects.LocationEvidence, error) {
	f.reviewer, f.reason, f.accessedAt = reviewer, reason, accessedAt
	return f.value, nil
}

func adminDependencies(deps Dependencies, evidence EvidenceReader) AdminDependencies {
	snapshot := deps.Reader.(*currentAttendanceReaderFake).resolved
	return AdminDependencies{
		Enrollment:   deps.Enrollment,
		Participants: &participantRepoFake{participant: snapshot.Participant},
		Shifts:       &shiftRepoFake{shift: snapshot.Shift, assignment: snapshot.Assignment},
		Workplaces:   &workplaceRepoFake{value: snapshot.Workplace},
		Attendance:   deps.Attendance,
		Clock:        deps.Clock,
		Evidence:     evidence,
	}
}

func attendanceWithClockIn(t *testing.T, deps Dependencies) aggregates.Attendance {
	t.Helper()
	snapshot := deps.Reader.(*currentAttendanceReaderFake).resolved
	value, err := aggregates.NewAttendance(snapshot.Participant, snapshot.Workplace, snapshot.Shift, snapshot.Assignment)
	if err != nil {
		t.Fatal(err)
	}
	evidence, _ := valueobjects.NewLocationEvidence(25, 121, 5)
	ref, _ := valueobjects.NewEvidenceRef("punch")
	location, err := snapshot.Workplace.AssessLocation(evidence, ref)
	if err != nil {
		t.Fatal(err)
	}
	punchID, _ := valueobjects.NewPunchID("punch")
	submissionID, _ := valueobjects.NewSubmissionID("submission")
	if _, err := value.RecordClockIn(punchID, submissionID, deps.Clock.Now(), location); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAdminInviteRequiresExpiryWithinTwentyFourHours(t *testing.T) {
	deps, _ := attendanceFixture(t)
	tokens := &tokenGeneratorFake{value: "one-time-code"}
	evidence := &evidenceReaderFake{}
	admin, err := NewAdminService(adminDependencies(deps, evidence), tokens)
	if err != nil {
		t.Fatal(err)
	}
	command := InviteEmployeeCommand{
		EmployeeID: "employee-new", EmployeeCode: "E2", DisplayName: "Employee Two",
		ExpiresAt: deps.Clock.Now().Add(24*time.Hour + time.Second),
	}
	if _, err := admin.InviteEmployee(context.Background(), command); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("long-lived invitation error=%v", err)
	}
	if tokens.used {
		t.Fatal("generated a token before validating invitation expiry")
	}
	command.ExpiresAt = deps.Clock.Now().Add(24 * time.Hour)
	if result, err := admin.InviteEmployee(context.Background(), command); err != nil || result.EnrollmentCode != "one-time-code" {
		t.Fatalf("bounded invitation=(%+v,%v)", result, err)
	}
}

func TestAdminInviteRejectsEmptyGeneratedTokenWithoutRepositoryCall(t *testing.T) {
	deps, _ := attendanceFixture(t)
	tokens := &tokenGeneratorFake{value: " \t "}
	admin, err := NewAdminService(adminDependencies(deps, &evidenceReaderFake{}), tokens)
	if err != nil {
		t.Fatal(err)
	}

	result, err := admin.InviteEmployee(context.Background(), InviteEmployeeCommand{
		EmployeeID: "employee-new", EmployeeCode: "E2", DisplayName: "Employee Two",
		ExpiresAt: deps.Clock.Now().Add(time.Hour),
	})
	if !errors.Is(err, ErrEnrollmentUnavailable) || result != (InviteEmployeeResult{}) {
		t.Fatalf("empty generated token result=(%+v,%v)", result, err)
	}
	if calls := deps.Enrollment.(*enrollmentFake).inviteCalls; calls != 0 {
		t.Fatalf("repository Invite calls=%d, want 0", calls)
	}
}

func TestAdminInviteNormalizesRepositoryErrors(t *testing.T) {
	tests := []struct {
		name string
		repo error
		want error
	}{
		{name: "not found", repo: repositories.ErrNotFound, want: ErrInvalidInput},
		{name: "conflict", repo: repositories.ErrConflict, want: ErrConflict},
		{name: "concurrent update", repo: repositories.ErrConcurrentUpdate, want: ErrConflict},
		{name: "canceled", repo: context.Canceled, want: context.Canceled},
		{name: "deadline", repo: context.DeadlineExceeded, want: context.DeadlineExceeded},
		{name: "unavailable", repo: errors.New("store unavailable"), want: ErrRepositoryUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps, _ := attendanceFixture(t)
			deps.Enrollment.(*enrollmentFake).err = test.repo
			admin, err := NewAdminService(adminDependencies(deps, &evidenceReaderFake{}), &tokenGeneratorFake{value: "code"})
			if err != nil {
				t.Fatal(err)
			}

			_, err = admin.InviteEmployee(context.Background(), InviteEmployeeCommand{
				EmployeeID: "employee-new", EmployeeCode: "E2", DisplayName: "Employee Two",
				ExpiresAt: deps.Clock.Now().Add(time.Hour),
			})
			if !errors.Is(err, test.want) {
				t.Fatalf("InviteEmployee error=%v, want %v", err, test.want)
			}
		})
	}
}

func TestAdminShowReviewAuditsRawEvidenceAccess(t *testing.T) {
	deps, participantID := attendanceFixture(t)
	deps.Attendance.(*attendanceRepoFake).value = attendanceWithClockIn(t, deps)
	deps.Attendance.(*attendanceRepoFake).findErr = nil
	evidenceValue, _ := valueobjects.NewLocationEvidence(25.1, 121.2, 7)
	evidence := &evidenceReaderFake{value: evidenceValue}
	admin, err := NewAdminService(adminDependencies(deps, evidence), &tokenGeneratorFake{value: "code"})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := admin.ShowReview(context.Background(), ShowReviewQuery{
		ParticipantID: string(participantID), AssignmentID: "assignment", Kind: ClockIn,
		IncludeLocation: true, Reviewer: "operator-1", AccessReason: "investigate anomaly",
	})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Location == nil || detail.Location.Latitude != 25.1 || evidence.reviewer != "operator-1" || evidence.reason != "investigate anomaly" || !evidence.accessedAt.Equal(deps.Clock.Now()) {
		t.Fatalf("detail=%+v audit=(%q,%q,%v)", detail, evidence.reviewer, evidence.reason, evidence.accessedAt)
	}
}

func TestAdminCreateShiftRequiresExistingWorkplace(t *testing.T) {
	deps, _ := attendanceFixture(t)
	adminDeps := adminDependencies(deps, &evidenceReaderFake{})
	adminDeps.Workplaces = &workplaceRepoFake{err: repositories.ErrNotFound}
	admin, err := NewAdminService(adminDeps, &tokenGeneratorFake{value: "code"})
	if err != nil {
		t.Fatal(err)
	}
	now := deps.Clock.Now()
	err = admin.CreateShift(context.Background(), CreateShiftCommand{
		ShiftInstanceID: "shift-new", WorkplaceID: "missing-workplace", Label: "Day Shift",
		ClockInOpensAt: now, ClockInClosesAt: now.Add(time.Hour), ClockOutOpensAt: now.Add(7 * time.Hour), ClockOutClosesAt: now.Add(8 * time.Hour), SubmissionOpensAt: now.Add(-time.Hour),
		SubmissionClosesAt: now.Add(9 * time.Hour), PolicyVersion: "shift-v1",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("missing workplace error=%v", err)
	}
	if adminDeps.Shifts.(*shiftRepoFake).saved {
		t.Fatal("saved shift with missing workplace")
	}
}

func TestAdminAssignEmployeeDerivesOpaqueAssignmentID(t *testing.T) {
	deps, _ := attendanceFixture(t)
	adminDeps := adminDependencies(deps, &evidenceReaderFake{})
	admin, err := NewAdminService(adminDeps, &tokenGeneratorFake{value: "code"})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.AssignEmployee(context.Background(), AssignEmployeeCommand{EmployeeID: "employee", ShiftInstanceID: "shift"}); err != nil {
		t.Fatal(err)
	}
	assignment := adminDeps.Shifts.(*shiftRepoFake).savedAssignment
	if !strings.HasPrefix(string(assignment.ID()), "assignment:v1:") || strings.Contains(string(assignment.ID()), "employee") {
		t.Fatalf("assignment ID is not opaque: %q", assignment.ID())
	}
}
