package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
)

func TestAttendanceMutationRequiresExplicitScopeAndApprovalBeforeRuntime(t *testing.T) {
	t.Parallel()
	called := 0
	deps := Dependencies{Attendance: func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		called++
		return &fakeAttendanceRuntime{}, nil
	}}
	getenv := attendanceTestEnvironment
	invalid := [][]string{
		{"employee", "deactivate", "--project", "project-1", "--environment", "production", "--employee-id", "employee-1"},
		{"employee", "deactivate", "--approve-write", "--environment", "production", "--employee-id", "employee-1"},
		{"employee", "deactivate", "--approve-write", "--project", "project-1", "--employee-id", "employee-1"},
		{"employee", "deactivate", "--approve-write", "--project", "project-1", "--environment", "prod", "--employee-id", "employee-1"},
	}
	for _, args := range invalid {
		var output bytes.Buffer
		if err := runAttendance(context.Background(), args, getenv, &output, deps); err == nil {
			t.Fatalf("accepted unsafe arguments %v", args)
		}
		if output.Len() != 0 {
			t.Fatalf("unsafe arguments produced output: %q", output.String())
		}
	}
	if called != 0 {
		t.Fatalf("runtime initialized %d times before validation", called)
	}
}

func TestAttendanceReviewLocationRequiresReasonBeforeRuntime(t *testing.T) {
	t.Parallel()
	called := false
	deps := Dependencies{Attendance: func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		called = true
		return &fakeAttendanceRuntime{}, nil
	}}
	args := []string{"review", "show", "--project", "project-1", "--environment", "staging", "--approve-write", "--participant-id", "participant-1", "--assignment-id", "assignment-1", "--kind", "clock_in", "--include-location"}
	if err := runAttendance(context.Background(), args, attendanceTestEnvironment, &bytes.Buffer{}, deps); err == nil || !strings.Contains(err.Error(), "--reason") {
		t.Fatalf("expected evidence-access reason error, got %v", err)
	}
	if called {
		t.Fatal("runtime initialized before raw evidence gate was validated")
	}
}

func TestAttendanceReviewLocationRequiresApprovalBeforeRuntime(t *testing.T) {
	t.Parallel()
	called := false
	deps := Dependencies{Attendance: func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		called = true
		return &fakeAttendanceRuntime{}, nil
	}}
	args := []string{"review", "show", "--project", "project-1", "--environment", "staging", "--participant-id", "participant-1", "--assignment-id", "assignment-1", "--kind", "clock_in", "--include-location", "--reviewer", "operator-1", "--reason", "investigate discrepancy"}
	if err := runAttendance(context.Background(), args, attendanceTestEnvironment, &bytes.Buffer{}, deps); err == nil || !strings.Contains(err.Error(), "--approve-write") {
		t.Fatalf("runAttendance(%v) error = %v, want --approve-write error", args, err)
	}
	if called {
		t.Fatal("runtime initialized before raw evidence approval was validated")
	}
}

func TestAttendanceReviewShowRemainsReadOnly(t *testing.T) {
	t.Parallel()
	runtime := &fakeAttendanceRuntime{}
	deps := Dependencies{Attendance: func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		return runtime, nil
	}}
	args := []string{"review", "show", "--project", "project-1", "--environment", "staging", "--participant-id", "participant-1", "--assignment-id", "assignment-1", "--kind", "clock_in"}
	if err := runAttendance(context.Background(), args, attendanceTestEnvironment, &bytes.Buffer{}, deps); err != nil {
		t.Fatalf("runAttendance(%v) error = %v, want nil", args, err)
	}
	if runtime.show.IncludeLocation {
		t.Fatalf("ShowReview query IncludeLocation = true, want false: %+v", runtime.show)
	}
}

func TestAttendanceReviewShowPassesAuditedQueryAndCloses(t *testing.T) {
	t.Parallel()
	runtime := &fakeAttendanceRuntime{detail: attendance.ReviewDetail{
		ReviewItem: attendance.ReviewItem{AttendanceID: "attendance-1", ParticipantID: "participant-1", AssignmentID: "assignment-1", PunchID: "punch-1", Kind: attendance.ClockIn, ReviewState: "review_required"},
		Location:   &attendance.LocationFix{Latitude: 25.04, Longitude: 121.56, AccuracyMeters: 8},
	}}
	var captured AttendanceRuntimeConfig
	deps := Dependencies{Attendance: func(ctx context.Context, config AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("runtime creation did not receive command deadline")
		}
		captured = config
		return runtime, nil
	}}
	args := []string{"review", "show", "--project", "project-1", "--environment", "staging", "--approve-write", "--participant-id", "participant-1", "--assignment-id", "assignment-1", "--kind", "clock_in", "--include-location", "--reviewer", "operator-1", "--reason", "investigate discrepancy"}
	var output bytes.Buffer
	if err := runAttendance(context.Background(), args, attendanceTestEnvironment, &output, deps); err != nil {
		t.Fatalf("run attendance review: %v", err)
	}
	if captured.ProjectID != "project-1" || captured.Environment != "staging" || captured.ActiveKeyID != "key-1" {
		t.Fatalf("unexpected runtime config: %+v", captured)
	}
	if !runtime.closed {
		t.Fatal("attendance runtime was not closed")
	}
	if !runtime.show.IncludeLocation || runtime.show.Reviewer != "operator-1" || runtime.show.AccessReason != "investigate discrepancy" {
		t.Fatalf("raw evidence query lost audit gate: %+v", runtime.show)
	}
	for _, marker := range []string{`"participant_id":"participant-1"`, `"latitude":25.04`, `"accuracy_meters":8`} {
		if !strings.Contains(output.String(), marker) {
			t.Fatalf("output missing %s: %s", marker, output.String())
		}
	}
}

func TestAttendanceCanceledParentDoesNotCreateRuntime(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	deps := Dependencies{Attendance: func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		called = true
		return &fakeAttendanceRuntime{}, nil
	}}
	err := runAttendance(ctx, []string{"review", "list", "--project", "project-1", "--environment", "staging"}, attendanceTestEnvironment, &bytes.Buffer{}, deps)
	if err != context.Canceled {
		t.Fatalf("got %v, want context canceled", err)
	}
	if called {
		t.Fatal("runtime initialized after root cancellation")
	}
}

func TestAttendanceProvisioningPassesOperatorDisplayLabels(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	runtime := &fakeAttendanceRuntime{}
	deps := Dependencies{Attendance: func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error) { return runtime, nil }}
	workplaceArgs := []string{
		"workplace", "upsert", "--project", "project-1", "--environment", "staging", "--approve-write",
		"--workplace-id", "workplace-1", "--display-name", "Taipei Office", "--latitude", "25.033",
		"--longitude", "121.565", "--radius-meters", "100", "--max-accuracy-meters", "50", "--policy-version", "location-v1",
	}
	if err := runAttendance(context.Background(), workplaceArgs, attendanceTestEnvironment, &bytes.Buffer{}, deps); err != nil {
		t.Fatal(err)
	}
	if runtime.workplace.DisplayName != "Taipei Office" {
		t.Fatalf("workplace command=%+v", runtime.workplace)
	}
	shiftArgs := []string{
		"shift", "create", "--project", "project-1", "--environment", "staging", "--approve-write",
		"--shift-id", "shift-1", "--workplace-id", "workplace-1", "--label", "Day Shift",
		"--clock-in-opens-at", now.Format(time.RFC3339), "--clock-in-closes-at", now.Add(time.Hour).Format(time.RFC3339),
		"--clock-out-opens-at", now.Add(7 * time.Hour).Format(time.RFC3339), "--clock-out-closes-at", now.Add(8 * time.Hour).Format(time.RFC3339),
		"--submission-opens-at", now.Add(-time.Hour).Format(time.RFC3339), "--submission-closes-at", now.Add(9 * time.Hour).Format(time.RFC3339),
		"--policy-version", "shift-v1",
	}
	if err := runAttendance(context.Background(), shiftArgs, attendanceTestEnvironment, &bytes.Buffer{}, deps); err != nil {
		t.Fatal(err)
	}
	if runtime.shift.Label != "Day Shift" {
		t.Fatalf("shift command=%+v", runtime.shift)
	}
}

func attendanceTestEnvironment(key string) string {
	switch key {
	case attendanceKeyIDEnv:
		return "key-1"
	case attendanceKeyMapEnv:
		return "key-1=projects/project-1/secrets/attendance-location-key/versions/1"
	default:
		return ""
	}
}

type fakeAttendanceRuntime struct {
	closed    bool
	show      attendance.ShowReviewQuery
	detail    attendance.ReviewDetail
	workplace attendance.UpsertWorkplaceCommand
	shift     attendance.CreateShiftCommand
}

func (f *fakeAttendanceRuntime) InviteEmployee(context.Context, attendance.InviteEmployeeCommand) (attendance.InviteEmployeeResult, error) {
	return attendance.InviteEmployeeResult{EnrollmentCode: "code", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (f *fakeAttendanceRuntime) DeactivateEmployee(context.Context, attendance.DeactivateEmployeeCommand) error {
	return nil
}

func (f *fakeAttendanceRuntime) UpsertWorkplace(_ context.Context, command attendance.UpsertWorkplaceCommand) error {
	f.workplace = command
	return nil
}
func (f *fakeAttendanceRuntime) CreateShift(_ context.Context, command attendance.CreateShiftCommand) error {
	f.shift = command
	return nil
}
func (f *fakeAttendanceRuntime) AssignEmployee(context.Context, attendance.AssignEmployeeCommand) error {
	return nil
}
func (f *fakeAttendanceRuntime) ListReviewRequired(context.Context, int) ([]attendance.ReviewItem, error) {
	return nil, nil
}
func (f *fakeAttendanceRuntime) ShowReview(_ context.Context, query attendance.ShowReviewQuery) (attendance.ReviewDetail, error) {
	f.show = query
	return f.detail, nil
}
func (f *fakeAttendanceRuntime) Review(context.Context, attendance.ReviewCommand) error { return nil }
func (f *fakeAttendanceRuntime) Close() error {
	f.closed = true
	return nil
}

var _ AttendanceRuntime = (*fakeAttendanceRuntime)(nil)
