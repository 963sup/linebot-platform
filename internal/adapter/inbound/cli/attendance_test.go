package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
)

func TestAttendanceMutationRequiresExplicitScopeAndApprovalBeforeRuntime(t *testing.T) {
	called := 0
	deps := Dependencies{Attendance: func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		called++
		return &fakeAttendanceRuntime{}, nil
	}}
	for _, args := range [][]string{
		{"employee", "deactivate", "--project", "project-1", "--environment", "production", "--employee-id", "employee-1"},
		{"employee", "deactivate", "--approve-write", "--environment", "production", "--employee-id", "employee-1"},
		{"employee", "deactivate", "--approve-write", "--project", "project-1", "--employee-id", "employee-1"},
	} {
		if err := runAttendance(context.Background(), args, func(string) string { return "" }, &bytes.Buffer{}, deps); err == nil {
			t.Fatalf("accepted unsafe arguments %v", args)
		}
	}
	if called != 0 {
		t.Fatalf("runtime initialized %d times before validation", called)
	}
}

func TestAttendanceEmployeeInviteUsesFirestoreOnlyRuntime(t *testing.T) {
	runtime := &fakeAttendanceRuntime{}
	var captured AttendanceRuntimeConfig
	deps := Dependencies{Attendance: func(ctx context.Context, config AttendanceRuntimeConfig) (AttendanceRuntime, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("runtime creation did not receive command deadline")
		}
		captured = config
		return runtime, nil
	}}
	args := []string{"employee", "invite", "--project", "project-1", "--environment", "staging", "--approve-write", "--employee-id", "employee-1", "--employee-code", "E001", "--display-name", "Employee", "--expires-at", "2099-01-01T00:00:00Z"}
	var output bytes.Buffer
	if err := runAttendance(context.Background(), args, func(key string) string {
		if key == attendanceDatabaseEnv {
			return "attendance"
		}
		return ""
	}, &output, deps); err != nil {
		t.Fatal(err)
	}
	if captured.ProjectID != "project-1" || captured.Environment != "staging" || captured.FirestoreDatabase != "attendance" {
		t.Fatalf("runtime config = %+v", captured)
	}
	if !runtime.closed || !strings.Contains(output.String(), "enrollment_code") {
		t.Fatalf("runtime/output = closed %v, %s", runtime.closed, output.String())
	}
}

func TestAttendanceRejectsRemovedAdministrationCommands(t *testing.T) {
	for _, command := range []string{"workplace", "shift", "review"} {
		err := runAttendance(context.Background(), []string{command, "list"}, func(string) string { return "" }, &bytes.Buffer{}, Dependencies{})
		if err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("%s error = %v", command, err)
		}
	}
}

type fakeAttendanceRuntime struct{ closed bool }

func (*fakeAttendanceRuntime) InviteEmployee(context.Context, attendance.InviteEmployeeCommand) (attendance.InviteEmployeeResult, error) {
	return attendance.InviteEmployeeResult{EnrollmentCode: "code"}, nil
}
func (*fakeAttendanceRuntime) DeactivateEmployee(context.Context, attendance.DeactivateEmployeeCommand) error {
	return nil
}
func (f *fakeAttendanceRuntime) Close() error { f.closed = true; return nil }

var _ AttendanceRuntime = (*fakeAttendanceRuntime)(nil)
