// Package repositories defines technology-free Attendance persistence contracts.
package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var (
	ErrNotFound           = errors.New("attendance repository record not found")
	ErrAttendanceNotFound = ErrNotFound
	ErrConflict           = errors.New("attendance repository uniqueness conflict")
	ErrConcurrentUpdate   = errors.New("attendance repository concurrent update")
)

// AttendanceRepository persists one participant/assignment aggregate atomically.
// SavePunch must compare-and-swap expectedRevision and commit the Aggregate plus
// separately encrypted, expiring raw evidence in the same transaction. SaveReview
// updates review history without creating new raw evidence. A retry of the same
// committed punch must remain distinguishable from a conflicting submission.
type AttendanceRepository interface {
	Find(context.Context, valueobjects.ParticipantID, valueobjects.ShiftAssignmentID) (aggregates.Attendance, error)
	ListReviewRequired(context.Context, int) ([]aggregates.Attendance, error)
	SavePunch(context.Context, aggregates.Attendance, uint64, valueobjects.PunchID, valueobjects.LocationEvidence, time.Time) error
	SaveReview(context.Context, aggregates.Attendance, uint64) error
}

// ParticipantRepository owns atomic participant lifecycle persistence.
type ParticipantRepository interface {
	FindByEmployeeID(context.Context, valueobjects.EmployeeID) (entities.Participant, error)
	FindByParticipantID(context.Context, valueobjects.ParticipantID) (entities.Participant, error)
	SaveParticipant(context.Context, entities.Participant) error
}

// EnrollmentRepository atomically consumes one invitation and binds identities.
// A retry by the same participant is idempotent. Reuse by another participant,
// or binding a participant already owned by another employee, must conflict.
type EnrollmentRepository interface {
	Invite(context.Context, entities.Participant, entities.EnrollmentInvitation) error
	ConsumeEnrollment(context.Context, valueobjects.EnrollmentCodeHash, valueobjects.ParticipantID, time.Time) (entities.Participant, valueobjects.EnrollmentOutcome, error)
}

// WorkplaceRepository owns atomic workplace definition persistence.
type WorkplaceRepository interface {
	FindWorkplace(context.Context, valueobjects.WorkplaceID) (entities.Workplace, error)
	SaveWorkplace(context.Context, entities.Workplace) error
}

// ShiftRepository atomically persists a shift and its participant assignments.
type ShiftRepository interface {
	FindShift(context.Context, valueobjects.ShiftInstanceID) (entities.ShiftInstance, error)
	SaveShift(context.Context, entities.ShiftInstance) error
	FindAssignment(context.Context, valueobjects.ShiftAssignmentID) (entities.ShiftAssignment, error)
	FindAssignmentForParticipant(context.Context, valueobjects.ParticipantID, valueobjects.ShiftInstanceID) (entities.ShiftAssignment, error)
	ListOpenAssignments(context.Context, valueobjects.ParticipantID, time.Time) ([]entities.ShiftAssignment, error)
	SaveAssignment(context.Context, entities.ShiftAssignment) error
}
