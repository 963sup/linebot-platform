// Package aggregates owns Attendance consistency boundaries.
package aggregates

import (
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var (
	ErrInvalidAttendanceDay = errors.New("invalid attendance day")
	ErrClockInRequired      = errors.New("clock in is required before clock out")
	ErrPunchOrdering        = errors.New("clock out must follow clock in")
)

// AttendanceDay is the consistency boundary for one participant and one workday.
type AttendanceDay struct {
	id            valueobjects.AttendanceDayID
	participantID valueobjects.ParticipantID
	workday       valueobjects.Workday
	clockIn       *entities.Punch
	clockOut      *entities.Punch
	revision      uint64
}

func NewAttendanceDay(id valueobjects.AttendanceDayID, participantID valueobjects.ParticipantID, workday valueobjects.Workday) (AttendanceDay, error) {
	if !id.Valid() || !participantID.Valid() || !workday.Valid() {
		return AttendanceDay{}, ErrInvalidAttendanceDay
	}
	return AttendanceDay{id: id, participantID: participantID, workday: workday}, nil
}

func RestoreAttendanceDay(id valueobjects.AttendanceDayID, participantID valueobjects.ParticipantID, workday valueobjects.Workday, clockIn, clockOut *entities.Punch, revision uint64) (AttendanceDay, error) {
	value, err := NewAttendanceDay(id, participantID, workday)
	if err != nil {
		return AttendanceDay{}, err
	}
	if revision == 0 || clockIn == nil || clockIn.Kind() != valueobjects.ClockIn ||
		(clockOut != nil && (clockOut.Kind() != valueobjects.ClockOut || !clockOut.RecordedAt().After(clockIn.RecordedAt()))) ||
		(clockOut == nil && revision != 1) || (clockOut != nil && revision != 2) {
		return AttendanceDay{}, ErrInvalidAttendanceDay
	}
	inCopy := *clockIn
	value.clockIn = &inCopy
	if clockOut != nil {
		outCopy := *clockOut
		value.clockOut = &outCopy
	}
	value.revision = revision
	return value, nil
}

func (a AttendanceDay) ID() valueobjects.AttendanceDayID          { return a.id }
func (a AttendanceDay) ParticipantID() valueobjects.ParticipantID { return a.participantID }
func (a AttendanceDay) Workday() valueobjects.Workday             { return a.workday }
func (a AttendanceDay) Revision() uint64                          { return a.revision }
func (a AttendanceDay) ClockIn() (entities.Punch, bool)           { return punchCopy(a.clockIn) }
func (a AttendanceDay) ClockOut() (entities.Punch, bool)          { return punchCopy(a.clockOut) }

func punchCopy(value *entities.Punch) (entities.Punch, bool) {
	if value == nil {
		return entities.Punch{}, false
	}
	return *value, true
}

func (a *AttendanceDay) RecordClockIn(id valueobjects.PunchID, serverTime time.Time, location valueobjects.LocationFix) (valueobjects.SubmissionOutcome, entities.Punch, error) {
	return a.record(id, valueobjects.ClockIn, serverTime, location)
}
func (a *AttendanceDay) RecordClockOut(id valueobjects.PunchID, serverTime time.Time, location valueobjects.LocationFix) (valueobjects.SubmissionOutcome, entities.Punch, error) {
	return a.record(id, valueobjects.ClockOut, serverTime, location)
}
func (a *AttendanceDay) record(id valueobjects.PunchID, kind valueobjects.PunchKind, serverTime time.Time, location valueobjects.LocationFix) (valueobjects.SubmissionOutcome, entities.Punch, error) {
	if a == nil || !a.id.Valid() || !kind.Valid() {
		return "", entities.Punch{}, ErrInvalidAttendanceDay
	}
	existing := a.clockIn
	if kind == valueobjects.ClockOut {
		existing = a.clockOut
	}
	if existing != nil {
		return valueobjects.SubmissionDuplicate, *existing, nil
	}
	if kind == valueobjects.ClockOut {
		if a.clockIn == nil {
			return "", entities.Punch{}, ErrClockInRequired
		}
		if !serverTime.After(a.clockIn.RecordedAt()) {
			return "", entities.Punch{}, ErrPunchOrdering
		}
	}
	punch, err := entities.NewPunch(id, kind, serverTime, location)
	if err != nil {
		return "", entities.Punch{}, err
	}
	if kind == valueobjects.ClockIn {
		a.clockIn = &punch
	} else {
		a.clockOut = &punch
	}
	a.revision++
	return valueobjects.SubmissionRecorded, punch, nil
}
