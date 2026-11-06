package attendance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type Service struct{ deps Dependencies }

const (
	maxEnrollmentCodeLength = 256
	rawEvidenceRetention    = 90 * 24 * time.Hour
)

func NewService(deps Dependencies) (*Service, error) {
	if deps.Verifier == nil || deps.Enrollment == nil || deps.Reader == nil || deps.Attendance == nil || deps.Clock == nil {
		return nil, errors.New("attendance dependencies are required")
	}
	return &Service{deps: deps}, nil
}

func (s *Service) Enroll(ctx context.Context, command EnrollCommand) (EnrollResult, error) {
	principal, err := s.principal(ctx, command.IDToken)
	if err != nil {
		return EnrollResult{}, err
	}
	now := s.deps.Clock.Now().UTC()
	code := strings.TrimSpace(command.Code)
	if code == "" {
		return EnrollResult{}, ErrEnrollmentInvalid
	}
	if !utf8.ValidString(code) || utf8.RuneCountInString(code) > maxEnrollmentCodeLength {
		return EnrollResult{}, ErrInvalidInput
	}
	codeHash := enrollmentCodeHash(code)
	participant, outcome, err := s.deps.Enrollment.ConsumeEnrollment(ctx, codeHash, principal.ParticipantID, now)
	if err != nil {
		return EnrollResult{}, repositoryError(ctx, err, ErrEnrollmentInvalid)
	}
	bound, ok := participant.ParticipantID()
	if !ok || bound != principal.ParticipantID || !participant.Active() {
		return EnrollResult{}, ErrEnrollmentInvalid
	}
	var status EnrollStatus
	switch outcome {
	case valueobjects.EnrollmentBound:
		status = EnrollCreated
	case valueobjects.EnrollmentAlreadyBound:
		status = EnrollAlreadyCompleted
	default:
		return EnrollResult{}, ErrRepositoryUnavailable
	}
	return EnrollResult{Status: status}, nil
}

func (s *Service) ListOptions(ctx context.Context, command OptionsCommand) (OptionsResult, error) {
	principal, err := s.principal(ctx, command.IDToken)
	if err != nil {
		return OptionsResult{}, err
	}
	now := s.deps.Clock.Now().UTC()
	snapshots, err := s.deps.Reader.ListCurrentOptions(ctx, principal.ParticipantID, now)
	if err != nil {
		return OptionsResult{}, currentReaderError(ctx, err, ErrRepositoryUnavailable)
	}
	result := OptionsResult{Options: make([]AttendanceOption, 0, len(snapshots))}
	for _, snapshot := range snapshots {
		if !currentSnapshotConsistent(snapshot, principal.ParticipantID, snapshot.Shift.ID()) || !snapshot.Participant.Active() || !snapshot.Shift.Active() || !snapshot.Workplace.Active() {
			return OptionsResult{}, ErrRepositoryUnavailable
		}
		allowedKinds := allowedKinds(snapshot.Attendance)
		if len(allowedKinds) == 0 {
			continue
		}
		label := snapshot.Shift.Label() + " · " + snapshot.Shift.StartsAt().Format(time.RFC3339) + " – " + snapshot.Shift.EndsAt().Format(time.RFC3339)
		result.Options = append(result.Options, AttendanceOption{ShiftInstanceID: string(snapshot.Shift.ID()), WorkplaceLabel: snapshot.Workplace.DisplayName(), Label: label, AllowedKinds: allowedKinds})
	}
	return result, nil
}

func allowedKinds(record *aggregates.Attendance) []PunchKind {
	if record == nil {
		return []PunchKind{ClockIn}
	}
	clockIn, hasClockIn := record.ClockIn()
	if !hasClockIn {
		return []PunchKind{ClockIn}
	}
	if clockIn.ReviewState() == valueobjects.ReviewVoided {
		return nil
	}
	if _, hasClockOut := record.ClockOut(); !hasClockOut {
		return []PunchKind{ClockOut}
	}
	return nil
}

func (s *Service) Punch(ctx context.Context, command PunchCommand) (PunchResult, error) {
	principal, err := s.principal(ctx, command.IDToken)
	if err != nil {
		return PunchResult{}, err
	}
	now := s.deps.Clock.Now().UTC()
	shiftID, err := valueobjects.NewShiftInstanceID(command.ShiftInstanceID)
	if err != nil || (command.Kind != ClockIn && command.Kind != ClockOut) {
		return PunchResult{}, ErrInvalidInput
	}
	evidence, err := valueobjects.NewLocationEvidence(command.Location.Latitude, command.Location.Longitude, command.Location.AccuracyMeters)
	if err != nil {
		return PunchResult{}, ErrInvalidInput
	}
	snapshot, err := s.deps.Reader.ResolvePunchContext(ctx, principal.ParticipantID, shiftID, now)
	if err != nil {
		return PunchResult{}, currentReaderError(ctx, err, ErrNotAssigned)
	}
	if !currentSnapshotConsistent(snapshot, principal.ParticipantID, shiftID) {
		return PunchResult{}, ErrRepositoryUnavailable
	}
	if !snapshot.Participant.Active() {
		return PunchResult{}, ErrNotEnrolled
	}
	if !snapshot.Shift.Active() || !snapshot.Workplace.Active() {
		return PunchResult{}, ErrNotAssigned
	}
	assignmentID := snapshot.Assignment.ID()
	var attendance aggregates.Attendance
	if snapshot.Attendance == nil {
		attendance, err = aggregates.NewAttendance(snapshot.Participant, snapshot.Workplace, snapshot.Shift, snapshot.Assignment)
		if err != nil {
			return PunchResult{}, ErrRepositoryUnavailable
		}
	} else {
		attendance = *snapshot.Attendance
	}
	punchID, _ := valueobjects.NewPunchID(opaqueID("punch", string(assignmentID), string(command.Kind)))
	ref, _ := valueobjects.NewEvidenceRef(string(punchID))
	assessment, err := attendance.Workplace().AssessLocation(evidence, ref)
	if err != nil {
		return PunchResult{}, ErrInvalidInput
	}
	baseRevision := attendance.Revision()
	submissionID, _ := valueobjects.NewSubmissionID(opaqueID("submission", string(assignmentID), string(command.Kind)))
	var outcome valueobjects.SubmissionOutcome
	if command.Kind == ClockIn {
		outcome, err = attendance.RecordClockIn(punchID, submissionID, now, assessment)
	} else {
		outcome, err = attendance.RecordClockOut(punchID, submissionID, now, assessment)
	}
	if err != nil {
		switch {
		case errors.Is(err, aggregates.ErrSubmissionWindowClosed):
			return PunchResult{}, ErrSubmissionClosed
		case errors.Is(err, aggregates.ErrClockInRequired), errors.Is(err, aggregates.ErrPunchOrdering):
			return PunchResult{}, ErrConflict
		}
		return PunchResult{}, err
	}
	if outcome == valueobjects.SubmissionRecorded {
		if err := s.deps.Attendance.SavePunch(ctx, attendance, baseRevision, punchID, evidence, now.Add(rawEvidenceRetention)); err != nil {
			if errors.Is(err, repositories.ErrConflict) || errors.Is(err, repositories.ErrConcurrentUpdate) {
				committed, findErr := s.deps.Attendance.Find(ctx, principal.ParticipantID, assignmentID)
				if findErr == nil {
					stored, ok := committed.ClockIn()
					if command.Kind == ClockOut {
						stored, ok = committed.ClockOut()
					}
					if ok && stored.ID() == punchID && stored.SubmissionID() == submissionID {
						attendance, outcome = committed, valueobjects.SubmissionDuplicate
					} else {
						return PunchResult{}, ErrConflict
					}
				} else {
					return PunchResult{}, repositoryError(ctx, findErr, ErrConflict)
				}
			} else {
				return PunchResult{}, repositoryError(ctx, err, ErrConflict)
			}
		}
	}
	punch, _ := attendance.ClockIn()
	if command.Kind == ClockOut {
		punch, _ = attendance.ClockOut()
	}
	reasons := reviewReasons(punch)
	if reasons == nil {
		reasons = []string{}
	}
	status := PunchRecorded
	if outcome == valueobjects.SubmissionDuplicate {
		status = PunchAlreadyRecorded
	}
	return PunchResult{AttendanceID: string(assignmentID), PunchID: string(punch.ID()), Kind: punch.Kind(), RecordedAt: punch.RecordedAt(), Status: status, ReviewState: string(punch.ReviewState()), ReviewReasons: reasons}, nil
}

func currentSnapshotConsistent(snapshot CurrentAttendanceSnapshot, participantID valueobjects.ParticipantID, shiftID valueobjects.ShiftInstanceID) bool {
	boundID, bound := snapshot.Participant.ParticipantID()
	if !bound || boundID != participantID || !snapshot.Assignment.ID().Valid() || snapshot.Assignment.EmployeeID() != snapshot.Participant.EmployeeID() || snapshot.Assignment.ShiftInstanceID() != shiftID || snapshot.Shift.ID() != shiftID || snapshot.Shift.WorkplaceID() != snapshot.Workplace.ID() {
		return false
	}
	if snapshot.Attendance == nil {
		return true
	}
	attendanceParticipantID, attendanceBound := snapshot.Attendance.Participant().ParticipantID()
	return attendanceBound && attendanceParticipantID == participantID &&
		snapshot.Attendance.Assignment().ID() == snapshot.Assignment.ID() &&
		snapshot.Attendance.Shift().ID() == snapshot.Shift.ID() &&
		snapshot.Attendance.Workplace().ID() == snapshot.Workplace.ID()
}

func currentReaderError(ctx context.Context, err, notFound error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, repositories.ErrNotFound):
		return notFound
	default:
		return ErrRepositoryUnavailable
	}
}

func repositoryError(ctx context.Context, err, notFound error) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, repositories.ErrNotFound):
		return notFound
	case errors.Is(err, repositories.ErrConflict), errors.Is(err, repositories.ErrConcurrentUpdate):
		return ErrConflict
	default:
		return ErrRepositoryUnavailable
	}
}

func (s *Service) principal(ctx context.Context, token string) (VerifiedPrincipal, error) {
	if ctx == nil {
		return VerifiedPrincipal{}, ErrUnauthenticated
	}
	if err := ctx.Err(); err != nil {
		return VerifiedPrincipal{}, err
	}
	if strings.TrimSpace(token) == "" {
		return VerifiedPrincipal{}, ErrUnauthenticated
	}
	principal, err := s.deps.Verifier.Verify(ctx, token)
	if err != nil {
		return VerifiedPrincipal{}, err
	}
	if !principal.ParticipantID.Valid() || principal.ValidatedAt.IsZero() {
		return VerifiedPrincipal{}, ErrUnauthenticated
	}
	return principal, nil
}

func opaqueID(kind string, parts ...string) string {
	h := sha256.New()
	_, _ = h.Write([]byte("attendance-" + kind + "-v1"))
	for _, part := range parts {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(part))
	}
	return kind + ":v1:" + hex.EncodeToString(h.Sum(nil))
}

var _ Actions = (*Service)(nil)
