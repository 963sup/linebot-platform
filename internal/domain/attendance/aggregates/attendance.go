// Package aggregates owns Attendance consistency boundaries.
package aggregates

import (
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var (
	ErrInvalidAttendance      = errors.New("invalid attendance aggregate")
	ErrSubmissionWindowClosed = errors.New("attendance submission window is closed")
	ErrClockInRequired        = errors.New("clock in is required before clock out")
	ErrPunchOrdering          = errors.New("clock out must follow clock in")
	ErrPunchNotFound          = errors.New("attendance punch not found")
)

// Attendance is the consistency boundary for one participant and one shift assignment.
// It owns at most one effective clock-in and one effective clock-out.
type Attendance struct {
	participant entities.Participant
	workplace   entities.Workplace
	shift       entities.ShiftInstance
	assignment  entities.ShiftAssignment
	clockIn     *entities.Punch
	clockOut    *entities.Punch
	revision    uint64
}

func NewAttendance(participant entities.Participant, workplace entities.Workplace, shift entities.ShiftInstance, assignment entities.ShiftAssignment) (Attendance, error) {
	participantID, bound := participant.ParticipantID()
	if !bound || !participantID.Valid() || !participant.Active() || !workplace.Active() || !shift.Active() || participant.EmployeeID() != assignment.EmployeeID() || workplace.ID() != shift.WorkplaceID() || shift.ID() != assignment.ShiftInstanceID() {
		return Attendance{}, ErrInvalidAttendance
	}
	return Attendance{participant: participant, workplace: workplace, shift: shift, assignment: assignment}, nil
}

func RestoreAttendance(participant entities.Participant, workplace entities.Workplace, shift entities.ShiftInstance, assignment entities.ShiftAssignment, clockIn, clockOut *entities.Punch, revision uint64) (Attendance, error) {
	value, err := NewAttendance(participant, workplace, shift, assignment)
	if err != nil {
		return Attendance{}, err
	}
	if clockIn != nil && clockIn.Kind() != valueobjects.ClockIn || clockOut != nil && clockOut.Kind() != valueobjects.ClockOut {
		return Attendance{}, ErrInvalidAttendance
	}
	if clockOut != nil && clockOut.ReviewState() != valueobjects.ReviewVoided && (clockIn == nil || clockIn.ReviewState() == valueobjects.ReviewVoided || !clockOut.EffectiveAt().After(clockIn.EffectiveAt())) {
		return Attendance{}, ErrPunchOrdering
	}
	if clockIn != nil {
		copy := *clockIn
		value.clockIn = &copy
	}
	if clockOut != nil {
		copy := *clockOut
		value.clockOut = &copy
	}
	value.revision = revision
	return value, nil
}

func (a Attendance) Participant() entities.Participant    { return a.participant }
func (a Attendance) Workplace() entities.Workplace        { return a.workplace }
func (a Attendance) Shift() entities.ShiftInstance        { return a.shift }
func (a Attendance) Assignment() entities.ShiftAssignment { return a.assignment }
func (a Attendance) Revision() uint64                     { return a.revision }
func (a Attendance) ClockIn() (entities.Punch, bool) {
	if a.clockIn == nil {
		return entities.Punch{}, false
	}
	return *a.clockIn, true
}
func (a Attendance) ClockOut() (entities.Punch, bool) {
	if a.clockOut == nil {
		return entities.Punch{}, false
	}
	return *a.clockOut, true
}

func (a *Attendance) RecordClockIn(id valueobjects.PunchID, submissionID valueobjects.SubmissionID, serverTime time.Time, location valueobjects.LocationAssessment) (valueobjects.SubmissionOutcome, error) {
	return a.record(id, valueobjects.ClockIn, submissionID, serverTime, location)
}

func (a *Attendance) RecordClockOut(id valueobjects.PunchID, submissionID valueobjects.SubmissionID, serverTime time.Time, location valueobjects.LocationAssessment) (valueobjects.SubmissionOutcome, error) {
	return a.record(id, valueobjects.ClockOut, submissionID, serverTime, location)
}

func (a *Attendance) record(id valueobjects.PunchID, kind valueobjects.PunchKind, submissionID valueobjects.SubmissionID, serverTime time.Time, location valueobjects.LocationAssessment) (valueobjects.SubmissionOutcome, error) {
	if a == nil {
		return "", ErrInvalidAttendance
	}
	existing := a.clockIn
	if kind == valueobjects.ClockOut {
		existing = a.clockOut
	}
	if existing != nil {
		return valueobjects.SubmissionDuplicate, nil
	}
	if !a.shift.AcceptsSubmission(serverTime) {
		return "", ErrSubmissionWindowClosed
	}
	if kind == valueobjects.ClockOut {
		if a.clockIn == nil || a.clockIn.ReviewState() == valueobjects.ReviewVoided {
			return "", ErrClockInRequired
		}
		if !serverTime.After(a.clockIn.EffectiveAt()) {
			return "", ErrPunchOrdering
		}
	}
	punch, err := entities.NewPunch(id, kind, submissionID, serverTime, location, a.shift.IsTimeAnomaly(kind, serverTime))
	if err != nil {
		return "", err
	}
	if kind == valueobjects.ClockIn {
		a.clockIn = &punch
	} else {
		a.clockOut = &punch
	}
	a.revision++
	return valueobjects.SubmissionRecorded, nil
}

func (a *Attendance) Confirm(kind valueobjects.PunchKind, reviewedAt time.Time, reviewer, reason string) error {
	punch, err := a.punch(kind)
	if err != nil {
		return err
	}
	if err := punch.Confirm(reviewedAt, reviewer, reason); err != nil {
		return err
	}
	a.revision++
	return nil
}

func (a *Attendance) Correct(kind valueobjects.PunchKind, effectiveAt time.Time, location valueobjects.LocationAssessment, provenance valueobjects.ReviewProvenance, reviewedAt time.Time, reviewer, reason string) error {
	punch, err := a.punch(kind)
	if err != nil {
		return err
	}
	candidate := *punch
	if err := candidate.Correct(effectiveAt, location, a.shift.IsTimeAnomaly(kind, effectiveAt), provenance, reviewedAt, reviewer, reason); err != nil {
		return err
	}
	if kind == valueobjects.ClockIn && a.clockOut != nil && a.clockOut.ReviewState() != valueobjects.ReviewVoided && !a.clockOut.EffectiveAt().After(candidate.EffectiveAt()) {
		return ErrPunchOrdering
	}
	if kind == valueobjects.ClockOut && a.clockIn != nil && a.clockIn.ReviewState() != valueobjects.ReviewVoided && !candidate.EffectiveAt().After(a.clockIn.EffectiveAt()) {
		return ErrPunchOrdering
	}
	*punch = candidate
	a.revision++
	return nil
}

func (a *Attendance) Void(kind valueobjects.PunchKind, reviewedAt time.Time, reviewer, reason string) error {
	punch, err := a.punch(kind)
	if err != nil {
		return err
	}
	if kind == valueobjects.ClockIn && a.clockOut != nil && a.clockOut.ReviewState() != valueobjects.ReviewVoided {
		return ErrPunchOrdering
	}
	if err := punch.Void(reviewedAt, reviewer, reason); err != nil {
		return err
	}
	a.revision++
	return nil
}

func (a *Attendance) punch(kind valueobjects.PunchKind) (*entities.Punch, error) {
	if a == nil {
		return nil, ErrPunchNotFound
	}
	if kind == valueobjects.ClockIn && a.clockIn != nil {
		return a.clockIn, nil
	}
	if kind == valueobjects.ClockOut && a.clockOut != nil {
		return a.clockOut, nil
	}
	return nil, ErrPunchNotFound
}
