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
	ErrNotAssigned           = errors.New("attendance shift is not assigned")
	ErrSubmissionClosed      = errors.New("attendance submission window is closed")
	ErrConflict              = errors.New("attendance operation conflicts with current state")
	ErrRepositoryUnavailable = errors.New("attendance repository unavailable")
)

type PunchKind = valueobjects.PunchKind

const (
	ClockIn  = valueobjects.ClockIn
	ClockOut = valueobjects.ClockOut
)

type EnrollCommand struct {
	IDToken string
	Code    string
}

type EnrollStatus string

const (
	EnrollCreated          EnrollStatus = "enrolled"
	EnrollAlreadyCompleted EnrollStatus = "already_enrolled"
)

type EnrollResult struct{ Status EnrollStatus }

type OptionsCommand struct{ IDToken string }

type AttendanceOption struct {
	ShiftInstanceID string
	WorkplaceLabel  string
	Label           string
	AllowedKinds    []PunchKind
}

type OptionsResult struct{ Options []AttendanceOption }

type LocationFix struct {
	Latitude       float64
	Longitude      float64
	AccuracyMeters float64
}

type PunchCommand struct {
	IDToken         string
	ShiftInstanceID string
	Kind            PunchKind
	Location        LocationFix
}

type PunchStatus string

const (
	PunchRecorded        PunchStatus = "recorded"
	PunchAlreadyRecorded PunchStatus = "already_recorded"
)

type PunchResult struct {
	AttendanceID  string
	PunchID       string
	Kind          PunchKind
	RecordedAt    time.Time
	Status        PunchStatus
	ReviewState   string
	ReviewReasons []string
}

// Actions is the inbound Application contract for the Mini App. HTTP adapters
// depend on this interface and never reconstruct attendance policy locally.
type Actions interface {
	Enroll(context.Context, EnrollCommand) (EnrollResult, error)
	ListOptions(context.Context, OptionsCommand) (OptionsResult, error)
	Punch(context.Context, PunchCommand) (PunchResult, error)
}
