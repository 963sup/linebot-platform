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

const (
	attendanceDatabaseEnv = "FIRESTORE_DATABASE"
	attendanceKeyIDEnv    = "ATTENDANCE_LOCATION_ACTIVE_KEY_ID"
	attendanceKeyMapEnv   = "ATTENDANCE_LOCATION_KEY_VERSIONS"
)

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

func (r *attendanceRuntimeAdapter) UpsertWorkplace(ctx context.Context, command attendance.UpsertWorkplaceCommand) error {
	return r.service.UpsertWorkplace(ctx, command)
}

func (r *attendanceRuntimeAdapter) CreateShift(ctx context.Context, command attendance.CreateShiftCommand) error {
	return r.service.CreateShift(ctx, command)
}

func (r *attendanceRuntimeAdapter) AssignEmployee(ctx context.Context, command attendance.AssignEmployeeCommand) error {
	return r.service.AssignEmployee(ctx, command)
}

func (r *attendanceRuntimeAdapter) ListReviewRequired(ctx context.Context, limit int) ([]attendance.ReviewItem, error) {
	return r.service.ListReviewRequired(ctx, limit)
}

func (r *attendanceRuntimeAdapter) ShowReview(ctx context.Context, query attendance.ShowReviewQuery) (attendance.ReviewDetail, error) {
	return r.service.ShowReview(ctx, query)
}

func (r *attendanceRuntimeAdapter) Review(ctx context.Context, command attendance.ReviewCommand) error {
	return r.service.Review(ctx, command)
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
		return errors.New("usage: ops attendance employee|workplace|shift|review ACTION [flags]")
	}
	switch args[0] {
	case "employee":
		return runAttendanceEmployee(parent, args[1], args[2:], getenv, output, deps)
	case "workplace":
		return runAttendanceWorkplace(parent, args[1], args[2:], getenv, output, deps)
	case "shift":
		return runAttendanceShift(parent, args[1], args[2:], getenv, output, deps)
	case "review":
		return runAttendanceReview(parent, args[1], args[2:], getenv, output, deps)
	default:
		return errors.New("unsupported attendance command")
	}
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

func runAttendanceWorkplace(parent context.Context, action string, args []string, getenv func(string) string, output io.Writer, deps Dependencies) error {
	if action != "upsert" {
		return errors.New("attendance workplace action must be upsert")
	}
	fs := flag.NewFlagSet("attendance workplace upsert", flag.ContinueOnError)
	common := addAttendanceCommonFlags(fs, true)
	id := fs.String("workplace-id", "", "workplace ID")
	displayName := fs.String("display-name", "", "operator-defined workplace display name")
	latitude := fs.Float64("latitude", 0, "workplace latitude")
	longitude := fs.Float64("longitude", 0, "workplace longitude")
	radius := fs.Float64("radius-meters", 0, "allowed radius in meters")
	accuracy := fs.Float64("max-accuracy-meters", 0, "maximum accepted accuracy in meters")
	policy := fs.String("policy-version", "", "location policy version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	config, err := attendanceRuntimeConfig(common, getenv, true)
	if err != nil {
		return err
	}
	if len(fs.Args()) != 0 || *id == "" || strings.TrimSpace(*displayName) == "" || !flagWasSet(fs, "latitude") || !flagWasSet(fs, "longitude") || *radius <= 0 || *accuracy <= 0 || strings.TrimSpace(*policy) == "" {
		return errors.New("workplace upsert requires ID, display name, coordinates, positive radius/accuracy and policy version")
	}
	return withAttendanceRuntime(parent, config, deps, func(ctx context.Context, runtime AttendanceRuntime) error {
		err := runtime.UpsertWorkplace(ctx, attendance.UpsertWorkplaceCommand{WorkplaceID: *id, DisplayName: *displayName, Latitude: *latitude, Longitude: *longitude, RadiusMeters: *radius, MaxAccuracyMeters: *accuracy, PolicyVersion: *policy})
		if err != nil {
			return err
		}
		return writeJSON(output, map[string]string{"status": "upserted", "workplace_id": *id})
	})
}

func runAttendanceShift(parent context.Context, action string, args []string, getenv func(string) string, output io.Writer, deps Dependencies) error {
	fs := flag.NewFlagSet("attendance shift "+action, flag.ContinueOnError)
	common := addAttendanceCommonFlags(fs, true)
	shiftID := fs.String("shift-id", "", "shift instance ID")
	label := fs.String("label", "", "operator-defined shift display label")
	workplaceID := fs.String("workplace-id", "", "workplace ID")
	employeeID := fs.String("employee-id", "", "employee ID")
	clockInOpensAt := fs.String("clock-in-opens-at", "", "clock-in review window open in RFC3339")
	clockInClosesAt := fs.String("clock-in-closes-at", "", "clock-in review window close in RFC3339")
	clockOutOpensAt := fs.String("clock-out-opens-at", "", "clock-out review window open in RFC3339")
	clockOutClosesAt := fs.String("clock-out-closes-at", "", "clock-out review window close in RFC3339")
	opensAt := fs.String("submission-opens-at", "", "submission window open in RFC3339")
	closesAt := fs.String("submission-closes-at", "", "submission window close in RFC3339")
	policy := fs.String("policy-version", "", "shift policy version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || (action != "create" && action != "assign") {
		return errors.New("attendance shift action must be create or assign")
	}
	config, err := attendanceRuntimeConfig(common, getenv, true)
	if err != nil {
		return err
	}
	var clockInOpen, clockInClose, clockOutOpen, clockOutClose, open, closeAt time.Time
	if action == "create" {
		var e1, e2, e3, e4, e5, e6 error
		clockInOpen, e1 = parseAttendanceTime("--clock-in-opens-at", *clockInOpensAt)
		clockInClose, e2 = parseAttendanceTime("--clock-in-closes-at", *clockInClosesAt)
		clockOutOpen, e3 = parseAttendanceTime("--clock-out-opens-at", *clockOutOpensAt)
		clockOutClose, e4 = parseAttendanceTime("--clock-out-closes-at", *clockOutClosesAt)
		open, e5 = parseAttendanceTime("--submission-opens-at", *opensAt)
		closeAt, e6 = parseAttendanceTime("--submission-closes-at", *closesAt)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || e6 != nil || *shiftID == "" || *workplaceID == "" || strings.TrimSpace(*label) == "" || strings.TrimSpace(*policy) == "" {
			return errors.New("shift create requires IDs, label, policy and all clock-in, clock-out and submission RFC3339 windows")
		}
	} else if *employeeID == "" || *shiftID == "" {
		return errors.New("shift assign requires --employee-id and --shift-id")
	}
	return withAttendanceRuntime(parent, config, deps, func(ctx context.Context, runtime AttendanceRuntime) error {
		switch action {
		case "create":
			if err := runtime.CreateShift(ctx, attendance.CreateShiftCommand{ShiftInstanceID: *shiftID, WorkplaceID: *workplaceID, Label: *label, ClockInOpensAt: clockInOpen, ClockInClosesAt: clockInClose, ClockOutOpensAt: clockOutOpen, ClockOutClosesAt: clockOutClose, SubmissionOpensAt: open, SubmissionClosesAt: closeAt, PolicyVersion: *policy}); err != nil {
				return err
			}
			return writeJSON(output, map[string]string{"status": "created", "shift_id": *shiftID})
		case "assign":
			if err := runtime.AssignEmployee(ctx, attendance.AssignEmployeeCommand{EmployeeID: *employeeID, ShiftInstanceID: *shiftID}); err != nil {
				return err
			}
			return writeJSON(output, map[string]string{"status": "assigned", "shift_id": *shiftID})
		default:
			return errors.New("unsupported shift action")
		}
	})
}

func runAttendanceReview(parent context.Context, action string, args []string, getenv func(string) string, output io.Writer, deps Dependencies) error {
	mutation := action == "confirm" || action == "correct" || action == "void"
	if action != "list" && action != "show" && !mutation {
		return errors.New("attendance review action must be list, show, confirm, correct or void")
	}
	fs := flag.NewFlagSet("attendance review "+action, flag.ContinueOnError)
	common := addAttendanceCommonFlags(fs, mutation || action == "show")
	limit := fs.Int("limit", 100, "maximum review records")
	participantID := fs.String("participant-id", "", "opaque participant ID")
	assignmentID := fs.String("assignment-id", "", "shift assignment ID")
	kind := fs.String("kind", "", "clock_in or clock_out")
	reviewer := fs.String("reviewer", "", "operator-supplied reviewer claim; Cloud Audit Logs are the authoritative identity record")
	reason := fs.String("reason", "", "review or evidence-access reason")
	includeLocation := fs.Bool("include-location", false, "decrypt and display short-lived raw location evidence")
	effectiveAt := fs.String("effective-at", "", "corrected effective time in RFC3339")
	distance := fs.Float64("distance-meters", 0, "corrected distance in meters")
	accuracy := fs.Float64("accuracy-meters", 0, "corrected accuracy in meters")
	outside := fs.Bool("outside-workplace", false, "corrected outside-workplace flag")
	lowAccuracy := fs.Bool("low-accuracy", false, "corrected low-accuracy flag")
	evidenceRef := fs.String("evidence-ref", "", "existing encrypted evidence reference")
	policy := fs.String("policy-version", "", "corrected policy version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return errors.New("unexpected attendance review arguments")
	}
	requiresApproval := mutation || (action == "show" && *includeLocation)
	config, err := attendanceRuntimeConfig(common, getenv, requiresApproval)
	if err != nil {
		return err
	}
	if action == "list" {
		if *limit <= 0 || *limit > 500 || *includeLocation || *participantID != "" || *assignmentID != "" || *kind != "" || *reason != "" || *reviewer != "" {
			return errors.New("review list accepts a limit from 1 to 500 and common flags")
		}
	} else {
		punchKind, kindErr := parsePunchKind(*kind)
		if kindErr != nil || *participantID == "" || *assignmentID == "" {
			return errors.New("review action requires participant, assignment and kind")
		}
		_ = punchKind
		if action == "show" && *includeLocation && (strings.TrimSpace(*reviewer) == "" || strings.TrimSpace(*reason) == "") {
			return errors.New("--include-location requires non-empty --reviewer and --reason")
		}
		if mutation && (*includeLocation || strings.TrimSpace(*reviewer) == "" || strings.TrimSpace(*reason) == "") {
			return errors.New("review mutation requires --reviewer and --reason and does not accept --include-location")
		}
		if action == "correct" {
			if _, timeErr := parseAttendanceTime("--effective-at", *effectiveAt); timeErr != nil || *accuracy <= 0 || *distance < 0 || *evidenceRef == "" || strings.TrimSpace(*policy) == "" {
				return errors.New("review correct requires effective time, location assessment, evidence reference and policy")
			}
		}
	}
	return withAttendanceRuntime(parent, config, deps, func(ctx context.Context, runtime AttendanceRuntime) error {
		if action == "list" {
			items, err := runtime.ListReviewRequired(ctx, *limit)
			if err != nil {
				return err
			}
			if items == nil {
				items = []attendance.ReviewItem{}
			}
			encoded := make([]attendanceReviewItemOutput, 0, len(items))
			for _, item := range items {
				encoded = append(encoded, newAttendanceReviewItemOutput(item))
			}
			return writeJSON(output, encoded)
		}
		punchKind, err := parsePunchKind(*kind)
		if err != nil {
			return err
		}
		if action == "show" {
			detail, err := runtime.ShowReview(ctx, attendance.ShowReviewQuery{ParticipantID: *participantID, AssignmentID: *assignmentID, Kind: punchKind, IncludeLocation: *includeLocation, Reviewer: *reviewer, AccessReason: *reason})
			if err != nil {
				return err
			}
			encoded := attendanceReviewDetailOutput{Item: newAttendanceReviewItemOutput(detail.ReviewItem)}
			if detail.Location != nil {
				encoded.Location = &attendanceLocationOutput{Latitude: detail.Location.Latitude, Longitude: detail.Location.Longitude, AccuracyMeters: detail.Location.AccuracyMeters}
			}
			return writeJSON(output, encoded)
		}
		command := attendance.ReviewCommand{ParticipantID: *participantID, AssignmentID: *assignmentID, Kind: punchKind, Action: action, Reviewer: *reviewer, Reason: *reason}
		if action == "correct" {
			at, _ := parseAttendanceTime("--effective-at", *effectiveAt)
			command.EffectiveAt, command.DistanceMeters, command.AccuracyMeters = at, *distance, *accuracy
			command.OutsideWorkplace, command.LowAccuracy = *outside, *lowAccuracy
			command.EvidenceRef, command.PolicyVersion = *evidenceRef, *policy
		}
		if err := runtime.Review(ctx, command); err != nil {
			return err
		}
		return writeJSON(output, map[string]string{"status": action, "assignment_id": *assignmentID, "kind": string(punchKind)})
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
	active, versions := getenv(attendanceKeyIDEnv), getenv(attendanceKeyMapEnv)
	if active == "" || versions == "" {
		return AttendanceRuntimeConfig{}, errors.New("Attendance encryption key configuration is required")
	}
	return AttendanceRuntimeConfig{ProjectID: *flags.project, Environment: *flags.environment, FirestoreDatabase: database, ActiveKeyID: active, KeyVersions: versions}, nil
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

func parsePunchKind(raw string) (attendance.PunchKind, error) {
	kind := attendance.PunchKind(raw)
	if kind != attendance.ClockIn && kind != attendance.ClockOut {
		return "", errors.New("--kind must be clock_in or clock_out")
	}
	return kind, nil
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(current *flag.Flag) {
		if current.Name == name {
			found = true
		}
	})
	return found
}

type attendanceInviteOutput struct {
	EnrollmentCode string    `json:"enrollment_code"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type attendanceReviewItemOutput struct {
	AttendanceID  string               `json:"attendance_id"`
	ParticipantID string               `json:"participant_id"`
	AssignmentID  string               `json:"assignment_id"`
	PunchID       string               `json:"punch_id"`
	Kind          attendance.PunchKind `json:"kind"`
	RecordedAt    time.Time            `json:"recorded_at"`
	ReviewState   string               `json:"review_state"`
	ReviewReasons []string             `json:"review_reasons"`
	EvidenceRef   string               `json:"evidence_ref"`
}

type attendanceReviewDetailOutput struct {
	Item     attendanceReviewItemOutput `json:"item"`
	Location *attendanceLocationOutput  `json:"location,omitempty"`
}

type attendanceLocationOutput struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters"`
}

func newAttendanceReviewItemOutput(item attendance.ReviewItem) attendanceReviewItemOutput {
	reasons := append([]string(nil), item.ReviewReasons...)
	if reasons == nil {
		reasons = []string{}
	}
	return attendanceReviewItemOutput{
		AttendanceID: item.AttendanceID, ParticipantID: item.ParticipantID, AssignmentID: item.AssignmentID,
		PunchID: item.PunchID, Kind: item.Kind, RecordedAt: item.RecordedAt, ReviewState: item.ReviewState,
		ReviewReasons: reasons, EvidenceRef: item.EvidenceRef,
	}
}
