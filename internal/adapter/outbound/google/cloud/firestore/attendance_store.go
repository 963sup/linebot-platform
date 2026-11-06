package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	sdk "cloud.google.com/go/firestore"
	attendanceapp "github.com/963sup/linebot-platform/internal/application/usecases/attendance"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"github.com/963sup/linebot-platform/internal/infrastructure/attendancecrypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	attendanceParticipantsCollection     = "attendance_participants_v1"
	attendanceParticipantBindings        = "attendance_participant_bindings_v1"
	attendanceEnrollmentTokensCollection = "attendance_enrollment_tokens_v1"
	attendanceWorkplacesCollection       = "attendance_workplaces_v1"
	attendanceShiftsCollection           = "attendance_shift_instances_v1"
	attendanceAssignmentsCollection      = "attendance_shift_assignments_v1"
	attendanceAssignmentGuardsCollection = "attendance_shift_assignment_guards_v1"
	attendanceRecordsCollection          = "attendance_records_v1"
	attendanceEvidenceCollection         = "attendance_location_evidence_v1"
	attendanceEvidenceAccessCollection   = "attendance_evidence_access_v1"
	attendanceReviewsCollection          = "attendance_review_actions_v1"
	attendanceSchemaVersion              = 1
	locationEvidenceRetention            = 90 * 24 * time.Hour
	maxParticipantAssignments            = 500
)

// AttendanceStore implements the Attendance repositories with transactional,
// versioned Firestore documents. It does not own the injected client or keyring.
type AttendanceStore struct {
	client  *sdk.Client
	keyring *attendancecrypto.Keyring
	now     func() time.Time
}

func NewAttendanceStore(client *sdk.Client, keyring *attendancecrypto.Keyring) (*AttendanceStore, error) {
	if client == nil || keyring == nil {
		return nil, errors.New("attendance store requires Firestore and encryption keyring")
	}
	return &AttendanceStore{client: client, keyring: keyring, now: time.Now}, nil
}

func (s *AttendanceStore) participantRef(id valueobjects.EmployeeID) *sdk.DocumentRef {
	return s.client.Collection(attendanceParticipantsCollection).Doc(attendanceDocumentID("employee", string(id)))
}
func (s *AttendanceStore) enrollmentRef(hash valueobjects.EnrollmentCodeHash) *sdk.DocumentRef {
	return s.client.Collection(attendanceEnrollmentTokensCollection).Doc(attendanceDocumentID("enrollment", string(hash)))
}
func (s *AttendanceStore) shiftRef(id valueobjects.ShiftInstanceID) *sdk.DocumentRef {
	return s.client.Collection(attendanceShiftsCollection).Doc(attendanceDocumentID("shift", string(id)))
}
func (s *AttendanceStore) assignmentRef(id valueobjects.ShiftAssignmentID) *sdk.DocumentRef {
	return s.client.Collection(attendanceAssignmentsCollection).Doc(attendanceDocumentID("assignment", string(id)))
}
func (s *AttendanceStore) assignmentGuardRef(employeeID valueobjects.EmployeeID, shiftID valueobjects.ShiftInstanceID) *sdk.DocumentRef {
	return s.client.Collection(attendanceAssignmentGuardsCollection).Doc(attendanceDocumentID("assignment-guard", string(employeeID), string(shiftID)))
}
func (s *AttendanceStore) attendanceRef(participantID valueobjects.ParticipantID, assignmentID valueobjects.ShiftAssignmentID) *sdk.DocumentRef {
	return s.client.Collection(attendanceRecordsCollection).Doc(attendanceDocumentID("attendance", string(participantID), string(assignmentID)))
}
func attendanceDocumentID(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(h, "%d:", len(part))
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func retentionExpired(expiresAt, now time.Time) bool {
	return !expiresAt.IsZero() && !expiresAt.After(now)
}

func attendanceExpiry(value time.Time) time.Time { return value.AddDate(5, 0, 0) }

func classifyAttendanceStoreError(ctx context.Context, err error) error {
	if err == nil || errors.Is(err, repositories.ErrNotFound) || errors.Is(err, repositories.ErrConflict) || errors.Is(err, repositories.ErrConcurrentUpdate) || errors.Is(err, ErrRepositoryCorrupt) {
		return err
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	switch status.Code(err) {
	case codes.NotFound:
		return repositories.ErrNotFound
	case codes.AlreadyExists:
		return repositories.ErrConflict
	case codes.Aborted, codes.FailedPrecondition:
		return repositories.ErrConcurrentUpdate
	default:
		return attendanceStoreError(ctx, "operation")
	}
}
func attendanceStoreError(ctx context.Context, operation string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: attendance %s failed", ErrRepositoryUnavailable, operation)
}

var _ repositories.AttendanceRepository = (*AttendanceStore)(nil)
var _ repositories.ParticipantRepository = (*AttendanceStore)(nil)
var _ repositories.EnrollmentRepository = (*AttendanceStore)(nil)
var _ repositories.WorkplaceRepository = (*AttendanceStore)(nil)
var _ repositories.ShiftRepository = (*AttendanceStore)(nil)
var _ attendanceapp.EvidenceReader = (*AttendanceStore)(nil)
