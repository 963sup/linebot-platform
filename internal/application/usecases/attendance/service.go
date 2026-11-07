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
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

const maxEnrollmentCodeLength = 256

var taipeiLocation = time.FixedZone("Asia/Taipei", 8*60*60)

type Service struct{ deps Dependencies }

func NewService(deps Dependencies) (*Service, error) {
	if deps.Verifier == nil || deps.Participants == nil || deps.Enrollment == nil || deps.Attendance == nil || deps.Clock == nil {
		return nil, errors.New("attendance dependencies are required")
	}
	return &Service{deps: deps}, nil
}

func (s *Service) Enroll(ctx context.Context, command EnrollCommand) (EnrollResult, error) {
	principal, err := s.principal(ctx, command.IDToken)
	if err != nil {
		return EnrollResult{}, err
	}
	code := strings.TrimSpace(command.Code)
	if code == "" {
		return EnrollResult{}, ErrEnrollmentInvalid
	}
	if !utf8.ValidString(code) || utf8.RuneCountInString(code) > maxEnrollmentCodeLength {
		return EnrollResult{}, ErrInvalidInput
	}
	participant, outcome, err := s.deps.Enrollment.ConsumeEnrollment(ctx, enrollmentCodeHash(code), principal.ParticipantID, s.deps.Clock.Now().UTC())
	if err != nil {
		return EnrollResult{}, repositoryError(ctx, err, ErrEnrollmentInvalid)
	}
	bound, ok := participant.ParticipantID()
	if !ok || bound != principal.ParticipantID || !participant.Active() {
		return EnrollResult{}, ErrEnrollmentInvalid
	}
	switch outcome {
	case valueobjects.EnrollmentBound:
		return EnrollResult{Status: EnrollCreated}, nil
	case valueobjects.EnrollmentAlreadyBound:
		return EnrollResult{Status: EnrollAlreadyCompleted}, nil
	default:
		return EnrollResult{}, ErrRepositoryUnavailable
	}
}

func (s *Service) GetToday(ctx context.Context, query TodayQuery) (TodayResult, error) {
	principal, err := s.enrolledPrincipal(ctx, query.IDToken)
	if err != nil {
		return TodayResult{}, err
	}
	workday, err := valueobjects.WorkdayAt(s.deps.Clock.Now().UTC(), taipeiLocation)
	if err != nil {
		return TodayResult{}, ErrRepositoryUnavailable
	}
	attendance, err := s.deps.Attendance.Find(ctx, principal.ParticipantID, workday)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return TodayResult{Workday: workday.String(), AllowedKinds: []PunchKind{ClockIn}}, nil
		}
		return TodayResult{}, repositoryError(ctx, err, ErrRepositoryUnavailable)
	}
	return todayResult(attendance), nil
}

func (s *Service) Punch(ctx context.Context, command PunchCommand) (PunchResult, error) {
	principal, err := s.enrolledPrincipal(ctx, command.IDToken)
	if err != nil {
		return PunchResult{}, err
	}
	if !command.Kind.Valid() {
		return PunchResult{}, ErrInvalidInput
	}
	location, err := valueobjects.NewLocationFix(command.Location.Latitude, command.Location.Longitude, command.Location.AccuracyMeters)
	if err != nil {
		return PunchResult{}, ErrInvalidInput
	}
	now := s.deps.Clock.Now().UTC()
	workday, err := valueobjects.WorkdayAt(now, taipeiLocation)
	if err != nil {
		return PunchResult{}, ErrRepositoryUnavailable
	}
	attendance, baseRevision, err := s.loadDay(ctx, principal.ParticipantID, workday)
	if err != nil {
		return PunchResult{}, err
	}
	punchID, _ := valueobjects.NewPunchID(opaqueID("punch", string(principal.ParticipantID), workday.String(), string(command.Kind)))
	var outcome valueobjects.SubmissionOutcome
	var punch entities.Punch
	if command.Kind == ClockIn {
		outcome, punch, err = attendance.RecordClockIn(punchID, now, location)
	} else {
		outcome, punch, err = attendance.RecordClockOut(punchID, now, location)
	}
	if err != nil {
		if errors.Is(err, aggregates.ErrClockInRequired) || errors.Is(err, aggregates.ErrPunchOrdering) {
			return PunchResult{}, ErrConflict
		}
		return PunchResult{}, err
	}
	if outcome == valueobjects.SubmissionRecorded {
		if err := s.deps.Attendance.Save(ctx, attendance, baseRevision); err != nil {
			if !errors.Is(err, repositories.ErrConflict) && !errors.Is(err, repositories.ErrConcurrentUpdate) {
				return PunchResult{}, repositoryError(ctx, err, ErrConflict)
			}
			committed, findErr := s.deps.Attendance.Find(ctx, principal.ParticipantID, workday)
			if findErr != nil {
				return PunchResult{}, repositoryError(ctx, findErr, ErrConflict)
			}
			stored, ok := committed.ClockIn()
			if command.Kind == ClockOut {
				stored, ok = committed.ClockOut()
			}
			if !ok || stored.ID() != punchID {
				return PunchResult{}, ErrConflict
			}
			attendance, punch, outcome = committed, stored, valueobjects.SubmissionDuplicate
		}
	}
	status := PunchRecorded
	if outcome == valueobjects.SubmissionDuplicate {
		status = PunchAlreadyRecorded
	}
	return PunchResult{
		AttendanceDayID: string(attendance.ID()),
		Workday:         attendance.Workday().String(),
		Punch:           punchView(punch),
		Status:          status,
	}, nil
}

func (s *Service) loadDay(ctx context.Context, participantID valueobjects.ParticipantID, workday valueobjects.Workday) (aggregates.AttendanceDay, uint64, error) {
	attendance, err := s.deps.Attendance.Find(ctx, participantID, workday)
	if err == nil {
		return attendance, attendance.Revision(), nil
	}
	if !errors.Is(err, repositories.ErrNotFound) {
		return aggregates.AttendanceDay{}, 0, repositoryError(ctx, err, ErrRepositoryUnavailable)
	}
	id, _ := valueobjects.NewAttendanceDayID(opaqueID("day", string(participantID), workday.String()))
	attendance, err = aggregates.NewAttendanceDay(id, participantID, workday)
	if err != nil {
		return aggregates.AttendanceDay{}, 0, ErrRepositoryUnavailable
	}
	return attendance, 0, nil
}

func (s *Service) enrolledPrincipal(ctx context.Context, token string) (VerifiedPrincipal, error) {
	principal, err := s.principal(ctx, token)
	if err != nil {
		return VerifiedPrincipal{}, err
	}
	participant, err := s.deps.Participants.FindByParticipantID(ctx, principal.ParticipantID)
	if err != nil {
		return VerifiedPrincipal{}, repositoryError(ctx, err, ErrNotEnrolled)
	}
	bound, ok := participant.ParticipantID()
	if !ok || bound != principal.ParticipantID || !participant.Active() {
		return VerifiedPrincipal{}, ErrNotEnrolled
	}
	return principal, nil
}

func (s *Service) principal(ctx context.Context, token string) (VerifiedPrincipal, error) {
	if ctx == nil || strings.TrimSpace(token) == "" {
		return VerifiedPrincipal{}, ErrUnauthenticated
	}
	if err := ctx.Err(); err != nil {
		return VerifiedPrincipal{}, err
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

func todayResult(attendance aggregates.AttendanceDay) TodayResult {
	result := TodayResult{Workday: attendance.Workday().String()}
	if punch, ok := attendance.ClockIn(); ok {
		view := punchView(punch)
		result.ClockIn = &view
	}
	if punch, ok := attendance.ClockOut(); ok {
		view := punchView(punch)
		result.ClockOut = &view
	}
	switch {
	case result.ClockIn == nil:
		result.AllowedKinds = []PunchKind{ClockIn}
	case result.ClockOut == nil:
		result.AllowedKinds = []PunchKind{ClockOut}
	default:
		result.AllowedKinds = []PunchKind{}
	}
	return result
}

func punchView(punch entities.Punch) PunchView {
	return PunchView{PunchID: string(punch.ID()), Kind: punch.Kind(), RecordedAt: punch.RecordedAt()}
}

func repositoryError(ctx context.Context, err, notFound error) error {
	if err == nil {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
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

func opaqueID(kind string, parts ...string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("attendance-" + kind + "-v1"))
	for _, part := range parts {
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(part))
	}
	return kind + ":v1:" + hex.EncodeToString(hash.Sum(nil))
}

var _ Actions = (*Service)(nil)
