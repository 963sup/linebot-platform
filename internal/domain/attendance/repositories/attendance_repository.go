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
	ErrNotFound         = errors.New("attendance repository record not found")
	ErrConflict         = errors.New("attendance repository uniqueness conflict")
	ErrConcurrentUpdate = errors.New("attendance repository concurrent update")
)

type AttendanceDayRepository interface {
	Find(context.Context, valueobjects.ParticipantID, valueobjects.Workday) (aggregates.AttendanceDay, error)
	Save(context.Context, aggregates.AttendanceDay, uint64) error
}

type ParticipantReader interface {
	FindByParticipantID(context.Context, valueobjects.ParticipantID) (entities.Participant, error)
}

type ParticipantRepository interface {
	ParticipantReader
	FindByEmployeeID(context.Context, valueobjects.EmployeeID) (entities.Participant, error)
	SaveParticipant(context.Context, entities.Participant) error
}

type EnrollmentRepository interface {
	Invite(context.Context, entities.Participant, entities.EnrollmentInvitation) error
	ConsumeEnrollment(context.Context, valueobjects.EnrollmentCodeHash, valueobjects.ParticipantID, time.Time) (entities.Participant, valueobjects.EnrollmentOutcome, error)
}
