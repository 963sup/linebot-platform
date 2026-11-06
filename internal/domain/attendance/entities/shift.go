package entities

import (
	"errors"
	"strings"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var ErrInvalidEntity = errors.New("invalid attendance entity")

type ShiftInstance struct {
	id                 valueobjects.ShiftInstanceID
	workplaceID        valueobjects.WorkplaceID
	startsAt           time.Time
	endsAt             time.Time
	submissionOpensAt  time.Time
	submissionClosesAt time.Time
	policyVersion      string
	label              string
	active             bool
	clockInWindow      valueobjects.TimeWindow
	clockOutWindow     valueobjects.TimeWindow
}

func DefineShiftInstance(id valueobjects.ShiftInstanceID, workplaceID valueobjects.WorkplaceID, label string, clockInWindow, clockOutWindow valueobjects.TimeWindow, submissionOpensAt, submissionClosesAt time.Time, policyVersion string) (ShiftInstance, error) {
	startsAt, endsAt := clockInWindow.OpensAt(), clockOutWindow.ClosesAt()
	if !id.Valid() || !workplaceID.Valid() || strings.TrimSpace(label) == "" || startsAt.IsZero() || !endsAt.After(startsAt) || clockInWindow.ClosesAt().After(clockOutWindow.ClosesAt()) || submissionOpensAt.IsZero() || submissionClosesAt.Before(submissionOpensAt) || submissionOpensAt.After(startsAt) || submissionClosesAt.Before(endsAt) || strings.TrimSpace(policyVersion) == "" {
		return ShiftInstance{}, ErrInvalidEntity
	}
	return ShiftInstance{id: id, workplaceID: workplaceID, startsAt: startsAt.UTC(), endsAt: endsAt.UTC(), submissionOpensAt: submissionOpensAt.UTC(), submissionClosesAt: submissionClosesAt.UTC(), policyVersion: policyVersion, label: label, active: true, clockInWindow: clockInWindow, clockOutWindow: clockOutWindow}, nil
}

func (s ShiftInstance) ID() valueobjects.ShiftInstanceID        { return s.id }
func (s ShiftInstance) WorkplaceID() valueobjects.WorkplaceID   { return s.workplaceID }
func (s ShiftInstance) StartsAt() time.Time                     { return s.startsAt }
func (s ShiftInstance) EndsAt() time.Time                       { return s.endsAt }
func (s ShiftInstance) SubmissionOpensAt() time.Time            { return s.submissionOpensAt }
func (s ShiftInstance) SubmissionClosesAt() time.Time           { return s.submissionClosesAt }
func (s ShiftInstance) PolicyVersion() string                   { return s.policyVersion }
func (s ShiftInstance) Label() string                           { return s.label }
func (s ShiftInstance) Active() bool                            { return s.active }
func (s ShiftInstance) ClockInWindow() valueobjects.TimeWindow  { return s.clockInWindow }
func (s ShiftInstance) ClockOutWindow() valueobjects.TimeWindow { return s.clockOutWindow }
func (s ShiftInstance) AcceptsSubmission(at time.Time) bool {
	return !at.IsZero() && !at.Before(s.submissionOpensAt) && !at.After(s.submissionClosesAt)
}
func (s ShiftInstance) IsTimeAnomaly(kind valueobjects.PunchKind, at time.Time) bool {
	if kind == valueobjects.ClockIn {
		return !s.clockInWindow.Contains(at)
	}
	return !s.clockOutWindow.Contains(at)
}
func (s *ShiftInstance) Deactivate() error {
	if s == nil || !s.id.Valid() {
		return ErrInvalidEntity
	}
	s.active = false
	return nil
}

type ShiftAssignment struct {
	id              valueobjects.ShiftAssignmentID
	employeeID      valueobjects.EmployeeID
	shiftInstanceID valueobjects.ShiftInstanceID
}

func NewShiftAssignment(id valueobjects.ShiftAssignmentID, employeeID valueobjects.EmployeeID, shiftInstanceID valueobjects.ShiftInstanceID) (ShiftAssignment, error) {
	if !id.Valid() || !employeeID.Valid() || !shiftInstanceID.Valid() {
		return ShiftAssignment{}, ErrInvalidEntity
	}
	return ShiftAssignment{id: id, employeeID: employeeID, shiftInstanceID: shiftInstanceID}, nil
}
func (a ShiftAssignment) ID() valueobjects.ShiftAssignmentID            { return a.id }
func (a ShiftAssignment) EmployeeID() valueobjects.EmployeeID           { return a.employeeID }
func (a ShiftAssignment) ShiftInstanceID() valueobjects.ShiftInstanceID { return a.shiftInstanceID }
