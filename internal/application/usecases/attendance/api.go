package attendance

import (
	"context"
	"errors"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var (
	ErrInvalidInput          = errors.New("invalid attendance input")
	ErrEnrollmentInvalid     = errors.New("attendance enrollment code is invalid")
	ErrEnrollmentUnavailable = errors.New("attendance enrollment is unavailable")
	ErrNotEnrolled           = errors.New("attendance participant is not enrolled")
	ErrConflict              = errors.New("attendance operation conflicts with current state")
	ErrRepositoryUnavailable = errors.New("attendance repository unavailable")
)

type PunchKind = valueobjects.PunchKind

const (
	ClockIn  = valueobjects.ClockIn
	ClockOut = valueobjects.ClockOut
)

type EnrollCommand struct{ IDToken, Code string }
type EnrollStatus string

const (
	EnrollCreated          EnrollStatus = "enrolled"
	EnrollAlreadyCompleted EnrollStatus = "already_enrolled"
)

type EnrollResult struct{ Status EnrollStatus }
type TodayQuery struct{ IDToken string }

type PunchView struct {
	PunchID    string
	Kind       PunchKind
	RecordedAt time.Time
}

type TodayResult struct {
	Workday      string
	ClockIn      *PunchView
	ClockOut     *PunchView
	AllowedKinds []PunchKind
}

type LocationFix struct{ Latitude, Longitude, AccuracyMeters float64 }
type PunchCommand struct {
	IDToken  string
	Kind     PunchKind
	Location LocationFix
}

type PunchStatus string

const (
	PunchRecorded        PunchStatus = "recorded"
	PunchAlreadyRecorded PunchStatus = "already_recorded"
)

type PunchResult struct {
	AttendanceDayID string
	Workday         string
	Punch           PunchView
	Status          PunchStatus
}

type Actions interface {
	Enroll(context.Context, EnrollCommand) (EnrollResult, error)
	GetToday(context.Context, TodayQuery) (TodayResult, error)
	Punch(context.Context, PunchCommand) (PunchResult, error)
}
