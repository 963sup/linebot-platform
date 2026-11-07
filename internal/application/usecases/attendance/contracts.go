// Package attendance orchestrates employee enrollment and workday attendance.
package attendance

import (
	"context"
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var (
	ErrUnauthenticated     = errors.New("attendance principal is not authenticated")
	ErrIdentityUnavailable = errors.New("attendance identity verification unavailable")
)

type VerifiedPrincipal struct {
	ParticipantID valueobjects.ParticipantID
	ValidatedAt   time.Time
}

type PrincipalVerifier interface {
	Verify(context.Context, string) (VerifiedPrincipal, error)
}

type Clock interface{ Now() time.Time }

type Dependencies struct {
	Verifier     PrincipalVerifier
	Participants repositories.ParticipantReader
	Enrollment   repositories.EnrollmentRepository
	Attendance   repositories.AttendanceDayRepository
	Clock        Clock
}

type AdminDependencies struct {
	Enrollment   repositories.EnrollmentRepository
	Participants repositories.ParticipantRepository
	Clock        Clock
}
