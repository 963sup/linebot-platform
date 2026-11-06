package attendance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type TokenGenerator interface{ Generate() (string, error) }
type InviteEmployeeCommand struct {
	EmployeeID, EmployeeCode, DisplayName string
	ExpiresAt                             time.Time
}
type InviteEmployeeResult struct {
	EnrollmentCode string
	ExpiresAt      time.Time
}
type DeactivateEmployeeCommand struct{ EmployeeID string }
type UpsertWorkplaceCommand struct {
	WorkplaceID, DisplayName                             string
	Latitude, Longitude, RadiusMeters, MaxAccuracyMeters float64
	PolicyVersion                                        string
}
type CreateShiftCommand struct {
	ShiftInstanceID, WorkplaceID, Label                                string
	ClockInOpensAt, ClockInClosesAt, ClockOutOpensAt, ClockOutClosesAt time.Time
	SubmissionOpensAt, SubmissionClosesAt                              time.Time
	PolicyVersion                                                      string
}
type AssignEmployeeCommand struct{ EmployeeID, ShiftInstanceID string }
type ReviewItem struct {
	AttendanceID, ParticipantID, AssignmentID, PunchID string
	Kind                                               PunchKind
	RecordedAt                                         time.Time
	ReviewState                                        string
	ReviewReasons                                      []string
	EvidenceRef                                        string
}
type ReviewDetail struct {
	ReviewItem
	Location *LocationFix
}
type ShowReviewQuery struct {
	ParticipantID, AssignmentID string
	Kind                        PunchKind
	IncludeLocation             bool
	Reviewer                    string
	AccessReason                string
}
type ReviewCommand struct {
	ParticipantID, AssignmentID    string
	Kind                           PunchKind
	Action, Reviewer, Reason       string
	EffectiveAt                    time.Time
	DistanceMeters, AccuracyMeters float64
	OutsideWorkplace, LowAccuracy  bool
	EvidenceRef, PolicyVersion     string
}
type AdminService struct {
	deps   AdminDependencies
	tokens TokenGenerator
}

func NewAdminService(deps AdminDependencies, tokens TokenGenerator) (*AdminService, error) {
	if deps.Enrollment == nil || deps.Participants == nil || deps.Shifts == nil || deps.Workplaces == nil || deps.Attendance == nil || deps.Clock == nil || deps.Evidence == nil || tokens == nil {
		return nil, errors.New("attendance admin dependencies are required")
	}
	return &AdminService{deps: deps, tokens: tokens}, nil
}
func (s *AdminService) ShowReview(ctx context.Context, q ShowReviewQuery) (ReviewDetail, error) {
	if err := validContext(ctx); err != nil {
		return ReviewDetail{}, err
	}
	pid, e1 := valueobjects.NewParticipantID(q.ParticipantID)
	aid, e2 := valueobjects.NewShiftAssignmentID(q.AssignmentID)
	if e1 != nil || e2 != nil || (q.Kind != ClockIn && q.Kind != ClockOut) {
		return ReviewDetail{}, ErrInvalidInput
	}
	v, err := s.deps.Attendance.Find(ctx, pid, aid)
	if err != nil {
		return ReviewDetail{}, repositoryError(ctx, err, ErrInvalidInput)
	}
	p, ok := v.ClockIn()
	if q.Kind == ClockOut {
		p, ok = v.ClockOut()
	}
	if !ok {
		return ReviewDetail{}, ErrInvalidInput
	}
	item := ReviewItem{AttendanceID: q.AssignmentID, ParticipantID: q.ParticipantID, AssignmentID: q.AssignmentID, PunchID: string(p.ID()), Kind: q.Kind, RecordedAt: p.RecordedAt(), ReviewState: string(p.ReviewState()), ReviewReasons: reviewReasons(p), EvidenceRef: string(p.Location().EvidenceRef())}
	detail := ReviewDetail{ReviewItem: item}
	if q.IncludeLocation {
		if strings.TrimSpace(q.Reviewer) == "" || strings.TrimSpace(q.AccessReason) == "" {
			return ReviewDetail{}, ErrInvalidInput
		}
		e, err := s.deps.Evidence.ReadEvidence(ctx, p.Location().EvidenceRef(), q.Reviewer, q.AccessReason, s.deps.Clock.Now().UTC())
		if err != nil {
			return ReviewDetail{}, err
		}
		detail.Location = &LocationFix{Latitude: e.Latitude(), Longitude: e.Longitude(), AccuracyMeters: e.AccuracyMeters()}
	}
	return detail, nil
}
func (s *AdminService) InviteEmployee(ctx context.Context, c InviteEmployeeCommand) (InviteEmployeeResult, error) {
	if err := validContext(ctx); err != nil {
		return InviteEmployeeResult{}, err
	}
	now := s.deps.Clock.Now().UTC()
	id, err := valueobjects.NewEmployeeID(c.EmployeeID)
	if err != nil || strings.TrimSpace(c.EmployeeCode) == "" || strings.TrimSpace(c.DisplayName) == "" || !c.ExpiresAt.After(now) || c.ExpiresAt.After(now.Add(24*time.Hour)) {
		return InviteEmployeeResult{}, ErrInvalidInput
	}
	p, err := entities.NewParticipant(id, c.EmployeeCode, c.DisplayName)
	if err != nil {
		return InviteEmployeeResult{}, ErrInvalidInput
	}
	token, err := s.tokens.Generate()
	if err != nil || strings.TrimSpace(token) == "" {
		return InviteEmployeeResult{}, ErrEnrollmentUnavailable
	}
	invitation, err := entities.NewEnrollmentInvitation(id, enrollmentCodeHash(token), c.ExpiresAt)
	if err != nil {
		return InviteEmployeeResult{}, ErrInvalidInput
	}
	if err = s.deps.Enrollment.Invite(ctx, p, invitation); err != nil {
		return InviteEmployeeResult{}, repositoryError(ctx, err, ErrInvalidInput)
	}
	return InviteEmployeeResult{token, c.ExpiresAt.UTC()}, nil
}
func (s *AdminService) DeactivateEmployee(ctx context.Context, c DeactivateEmployeeCommand) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	id, err := valueobjects.NewEmployeeID(c.EmployeeID)
	if err != nil {
		return ErrInvalidInput
	}
	p, err := s.deps.Participants.FindByEmployeeID(ctx, id)
	if err != nil {
		return repositoryError(ctx, err, ErrInvalidInput)
	}
	if err = p.Deactivate(s.deps.Clock.Now()); err != nil {
		return err
	}
	return repositoryError(ctx, s.deps.Participants.SaveParticipant(ctx, p), ErrInvalidInput)
}
func (s *AdminService) UpsertWorkplace(ctx context.Context, c UpsertWorkplaceCommand) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	id, e1 := valueobjects.NewWorkplaceID(c.WorkplaceID)
	point, e2 := valueobjects.NewGeoPoint(c.Latitude, c.Longitude)
	if e1 != nil || e2 != nil {
		return ErrInvalidInput
	}
	w, err := entities.DefineWorkplace(id, c.DisplayName, point, c.RadiusMeters, c.MaxAccuracyMeters, c.PolicyVersion)
	if err != nil {
		return ErrInvalidInput
	}
	return repositoryError(ctx, s.deps.Workplaces.SaveWorkplace(ctx, w), ErrInvalidInput)
}
func (s *AdminService) CreateShift(ctx context.Context, c CreateShiftCommand) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	id, e1 := valueobjects.NewShiftInstanceID(c.ShiftInstanceID)
	wid, e2 := valueobjects.NewWorkplaceID(c.WorkplaceID)
	if e1 != nil || e2 != nil {
		return ErrInvalidInput
	}
	workplace, err := s.deps.Workplaces.FindWorkplace(ctx, wid)
	if err != nil {
		return repositoryError(ctx, err, ErrInvalidInput)
	}
	if !workplace.Active() {
		return ErrConflict
	}
	clockInWindow, clockInErr := valueobjects.NewTimeWindow(c.ClockInOpensAt, c.ClockInClosesAt)
	clockOutWindow, clockOutErr := valueobjects.NewTimeWindow(c.ClockOutOpensAt, c.ClockOutClosesAt)
	if clockInErr != nil || clockOutErr != nil {
		return ErrInvalidInput
	}
	v, err := entities.DefineShiftInstance(id, wid, c.Label, clockInWindow, clockOutWindow, c.SubmissionOpensAt, c.SubmissionClosesAt, c.PolicyVersion)
	if err != nil {
		return ErrInvalidInput
	}
	return repositoryError(ctx, s.deps.Shifts.SaveShift(ctx, v), ErrInvalidInput)
}
func (s *AdminService) AssignEmployee(ctx context.Context, c AssignEmployeeCommand) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	eid, e1 := valueobjects.NewEmployeeID(c.EmployeeID)
	sid, e2 := valueobjects.NewShiftInstanceID(c.ShiftInstanceID)
	if e1 != nil || e2 != nil {
		return ErrInvalidInput
	}
	aid, _ := valueobjects.NewShiftAssignmentID(opaqueID("assignment", string(eid), string(sid)))
	p, err := s.deps.Participants.FindByEmployeeID(ctx, eid)
	if err != nil {
		return repositoryError(ctx, err, ErrInvalidInput)
	}
	if !p.Active() {
		return ErrConflict
	}
	if _, err = s.deps.Shifts.FindShift(ctx, sid); err != nil {
		return repositoryError(ctx, err, ErrInvalidInput)
	}
	a, err := entities.NewShiftAssignment(aid, eid, sid)
	if err != nil {
		return ErrInvalidInput
	}
	return repositoryError(ctx, s.deps.Shifts.SaveAssignment(ctx, a), ErrInvalidInput)
}
func (s *AdminService) ListReviewRequired(ctx context.Context, limit int) ([]ReviewItem, error) {
	if err := validContext(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		return nil, ErrInvalidInput
	}
	values, err := s.deps.Attendance.ListReviewRequired(ctx, limit)
	if err != nil {
		return nil, repositoryError(ctx, err, ErrInvalidInput)
	}
	var out []ReviewItem
	for _, v := range values {
		for _, kind := range []PunchKind{ClockIn, ClockOut} {
			p, ok := v.ClockIn()
			if kind == ClockOut {
				p, ok = v.ClockOut()
			}
			if ok && p.ReviewState() == valueobjects.ReviewRequired {
				participantID, _ := v.Participant().ParticipantID()
				out = append(out, ReviewItem{AttendanceID: string(v.Assignment().ID()), ParticipantID: string(participantID), AssignmentID: string(v.Assignment().ID()), PunchID: string(p.ID()), Kind: kind, RecordedAt: p.RecordedAt(), ReviewState: string(p.ReviewState()), ReviewReasons: reviewReasons(p), EvidenceRef: string(p.Location().EvidenceRef())})
			}
		}
	}
	return out, nil
}
func (s *AdminService) Review(ctx context.Context, c ReviewCommand) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	pid, e1 := valueobjects.NewParticipantID(c.ParticipantID)
	aid, e2 := valueobjects.NewShiftAssignmentID(c.AssignmentID)
	if e1 != nil || e2 != nil {
		return ErrInvalidInput
	}
	v, err := s.deps.Attendance.Find(ctx, pid, aid)
	if err != nil {
		return repositoryError(ctx, err, ErrInvalidInput)
	}
	rev := v.Revision()
	now := s.deps.Clock.Now()
	switch c.Action {
	case "confirm":
		err = v.Confirm(c.Kind, now, c.Reviewer, c.Reason)
	case "void":
		err = v.Void(c.Kind, now, c.Reviewer, c.Reason)
	case "correct":
		ref, e := valueobjects.NewEvidenceRef(c.EvidenceRef)
		a, e2 := valueobjects.NewLocationAssessment(c.DistanceMeters, c.AccuracyMeters, c.OutsideWorkplace, c.LowAccuracy, ref, c.PolicyVersion)
		if e != nil || e2 != nil {
			return ErrInvalidInput
		}
		err = v.Correct(c.Kind, c.EffectiveAt, a, valueobjects.ReviewProvenanceManualOverride, now, c.Reviewer, c.Reason)
	default:
		return ErrInvalidInput
	}
	if err != nil {
		return err
	}
	return repositoryError(ctx, s.deps.Attendance.SaveReview(ctx, v, rev), ErrInvalidInput)
}
func enrollmentCodeHash(code string) valueobjects.EnrollmentCodeHash {
	sum := sha256.Sum256([]byte("attendance-enrollment-v1\x00" + code))
	v, _ := valueobjects.NewEnrollmentCodeHash("v1:" + hex.EncodeToString(sum[:]))
	return v
}
func validContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidInput
	}
	return ctx.Err()
}
func reviewReasons(p entities.Punch) []string {
	var r []string
	if p.Location().OutsideWorkplace() {
		r = append(r, "outside_workplace")
	}
	if p.Location().LowAccuracy() {
		r = append(r, "low_accuracy")
	}
	if p.TimeAnomaly() {
		r = append(r, "time_anomaly")
	}
	return r
}
