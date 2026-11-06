// Package attendance orchestrates employee enrollment and attendance without
// exposing provider identity or persistence representations.
package attendance

import (
	"context"
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var (
	ErrUnauthenticated     = errors.New("attendance principal is not authenticated")
	ErrIdentityUnavailable = errors.New("attendance identity verification unavailable")
)

// VerifiedPrincipal contains only the opaque, application-safe participant
// identity derived by an adapter and proof time. Provider subjects never cross
// this boundary.
type VerifiedPrincipal struct {
	ParticipantID valueobjects.ParticipantID
	ValidatedAt   time.Time
}

type PrincipalVerifier interface {
	Verify(context.Context, string) (VerifiedPrincipal, error)
}

type Clock interface{ Now() time.Time }
type EvidenceReader interface {
	ReadEvidence(context.Context, valueobjects.EvidenceRef, string, string, time.Time) (valueobjects.LocationEvidence, error)
}

// CurrentAttendanceSnapshot is the coherent domain state needed to decide one
// current attendance option or punch. A nil Attendance means the slot has no
// attendance record yet.
type CurrentAttendanceSnapshot struct {
	Participant entities.Participant
	Assignment  entities.ShiftAssignment
	Shift       entities.ShiftInstance
	Workplace   entities.Workplace
	Attendance  *aggregates.Attendance
}

// CurrentAttendanceReader loads the current cross-repository view in one
// adapter call so Application does not orchestrate persistence joins.
type CurrentAttendanceReader interface {
	ResolvePunchContext(context.Context, valueobjects.ParticipantID, valueobjects.ShiftInstanceID, time.Time) (CurrentAttendanceSnapshot, error)
	ListCurrentOptions(context.Context, valueobjects.ParticipantID, time.Time) ([]CurrentAttendanceSnapshot, error)
}

type Dependencies struct {
	Verifier   PrincipalVerifier
	Enrollment repositories.EnrollmentRepository
	Reader     CurrentAttendanceReader
	Attendance repositories.AttendanceRepository
	Clock      Clock
}
type AdminDependencies struct {
	Enrollment   repositories.EnrollmentRepository
	Participants repositories.ParticipantRepository
	Shifts       repositories.ShiftRepository
	Workplaces   repositories.WorkplaceRepository
	Attendance   repositories.AttendanceRepository
	Clock        Clock
	Evidence     EvidenceReader
}
