package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
)

const attendanceCommandTimeout = 20 * time.Second

const attendanceDatabaseEnv = "FIRESTORE_DATABASE"

type attendanceRuntimeAdapter struct {
	service *attendance.AdminService
	close   func() error
}

// NewAttendanceRuntime adapts the Attendance Application service to the
// operator CLI's narrow lifecycle contract. The composition root supplies the
// close function for resources it owns.
func NewAttendanceRuntime(service *attendance.AdminService, close func() error) (AttendanceRuntime, error) {
	if service == nil || close == nil {
		return nil, errors.New("Attendance CLI runtime dependencies are required")
	}
	return &attendanceRuntimeAdapter{service: service, close: close}, nil
}

func (r *attendanceRuntimeAdapter) InviteEmployee(ctx context.Context, command attendance.InviteEmployeeCommand) (attendance.InviteEmployeeResult, error) {
	return r.service.InviteEmployee(ctx, command)
}

func (r *attendanceRuntimeAdapter) DeactivateEmployee(ctx context.Context, command attendance.DeactivateEmployeeCommand) error {
	return r.service.DeactivateEmployee(ctx, command)
}

func (r *attendanceRuntimeAdapter) Close() error { return r.close() }

var _ AttendanceRuntime = (*attendanceRuntimeAdapter)(nil)

type attendanceCommonFlags struct {
	project      *string
	environment  *string
	approveWrite *bool
}

func runAttendance(parent context.Context, args []string, getenv func(string) string, output io.Writer, deps Dependencies) error {
	if parent == nil || getenv == nil || output == nil {
		return errors.New("attendance command dependencies are required")
	}
	if err := parent.Err(); err != nil {
		return err
	}
	if len(args) < 2 {
		return errors.New("usage: ops attendance employee invite|deactivate [flags]")
	}
	if args[0] != "employee" {
		return errors.New("unsupported attendance command")
	}
	return runAttendanceEmployee(parent, args[1], args[2:], getenv, output, deps)
}

func runAttendanceEmployee(parent context.Context, action string, args []string, getenv func(string) string, output io.Writer, deps Dependencies) error {
	fs := flag.NewFlagSet("attendance employee "+action, flag.ContinueOnError)
	common := addAttendanceCommonFlags(fs, true)
	employeeID := fs.String("employee-id", "", "stable employee ID")
	employeeCode := fs.String("employee-code", "", "organization employee code")
	displayName := fs.String("display-name", "", "employee display name")
	expiresAt := fs.String("expires-at", "", "enrollment expiry in RFC3339")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || (action != "invite" && action != "deactivate") {
		return errors.New("attendance employee action must be invite or deactivate")
	}
	config, err := attendanceRuntimeConfig(common, getenv, true)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*employeeID) == "" {
		return errors.New("--employee-id is required")
	}
	var expiry time.Time
	if action == "invite" {
		expiry, err = parseAttendanceTime("--expires-at", *expiresAt)
		if err != nil || strings.TrimSpace(*employeeCode) == "" || strings.TrimSpace(*displayName) == "" {
			return errors.New("invite requires --employee-code, --display-name and RFC3339 --expires-at")
		}
	} else if *employeeCode != "" || *displayName != "" || *expiresAt != "" {
		return errors.New("deactivate accepts only --employee-id")
	}
	return withAttendanceRuntime(parent, config, deps, func(ctx context.Context, runtime AttendanceRuntime) error {
		switch action {
		case "invite":
			result, err := runtime.InviteEmployee(ctx, attendance.InviteEmployeeCommand{EmployeeID: *employeeID, EmployeeCode: *employeeCode, DisplayName: *displayName, ExpiresAt: expiry})
			if err != nil {
				return err
			}
			return writeJSON(output, attendanceInviteOutput{EnrollmentCode: result.EnrollmentCode, ExpiresAt: result.ExpiresAt})
		case "deactivate":
			if err := runtime.DeactivateEmployee(ctx, attendance.DeactivateEmployeeCommand{EmployeeID: *employeeID}); err != nil {
				return err
			}
			return writeJSON(output, map[string]string{"status": "deactivated"})
		default:
			return errors.New("unsupported employee action")
		}
	})
}

func addAttendanceCommonFlags(fs *flag.FlagSet, mutation bool) attendanceCommonFlags {
	flags := attendanceCommonFlags{
		project:     fs.String("project", "", "explicit Google Cloud project ID"),
		environment: fs.String("environment", "", "explicit local, staging or production environment"),
	}
	if mutation {
		flags.approveWrite = fs.Bool("approve-write", false, "explicitly authorize the write")
	}
	return flags
}

func attendanceRuntimeConfig(flags attendanceCommonFlags, getenv func(string) string, mutation bool) (AttendanceRuntimeConfig, error) {
	if flags.project == nil || flags.environment == nil || strings.TrimSpace(*flags.project) == "" {
		return AttendanceRuntimeConfig{}, errors.New("--project and --environment are required")
	}
	switch *flags.environment {
	case "local", "staging", "production":
	default:
		return AttendanceRuntimeConfig{}, errors.New("--environment must be local, staging or production")
	}
	if mutation && (flags.approveWrite == nil || !*flags.approveWrite) {
		return AttendanceRuntimeConfig{}, errors.New("attendance mutation requires --approve-write")
	}
	database := getenv(attendanceDatabaseEnv)
	if database == "" {
		database = "(default)"
	}
	return AttendanceRuntimeConfig{ProjectID: *flags.project, Environment: *flags.environment, FirestoreDatabase: database}, nil
}

func withAttendanceRuntime(parent context.Context, config AttendanceRuntimeConfig, deps Dependencies, operation func(context.Context, AttendanceRuntime) error) (result error) {
	if deps.Attendance == nil {
		return errors.New("attendance runtime is not configured")
	}
	ctx, cancel := context.WithTimeout(parent, attendanceCommandTimeout)
	defer cancel()
	runtime, err := deps.Attendance(ctx, config)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, runtime.Close()) }()
	return operation(ctx, runtime)
}

func parseAttendanceTime(name, raw string) (time.Time, error) {
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil || value.IsZero() {
		return time.Time{}, errors.New(name + " must be RFC3339")
	}
	return value.UTC(), nil
}

type attendanceInviteOutput struct {
	EnrollmentCode string    `json:"enrollment_code"`
	ExpiresAt      time.Time `json:"expires_at"`
}
