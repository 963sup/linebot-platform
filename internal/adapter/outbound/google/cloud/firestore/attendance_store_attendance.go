package firestore

import (
	"context"
	"errors"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AttendanceStore) attendanceDayRef(participantID valueobjects.ParticipantID, workday valueobjects.Workday) *sdk.DocumentRef {
	return s.client.Collection(attendanceDaysCollection).Doc(attendanceDocumentID("attendance-day", string(participantID), workday.String()))
}

func (s *AttendanceStore) Find(ctx context.Context, participantID valueobjects.ParticipantID, workday valueobjects.Workday) (aggregates.AttendanceDay, error) {
	if ctx == nil || !participantID.Valid() || !workday.Valid() {
		return aggregates.AttendanceDay{}, errors.New("valid attendance day identity is required")
	}
	snapshot, err := s.attendanceDayRef(participantID, workday).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return aggregates.AttendanceDay{}, repositories.ErrNotFound
	}
	if err != nil {
		return aggregates.AttendanceDay{}, attendanceStoreError(ctx, "read attendance day")
	}
	var stored attendanceDayDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return aggregates.AttendanceDay{}, ErrRepositoryCorrupt
	}
	value, err := restoreAttendanceDay(stored)
	if err != nil || value.ParticipantID() != participantID || value.Workday() != workday {
		return aggregates.AttendanceDay{}, ErrRepositoryCorrupt
	}
	return value, nil
}

func (s *AttendanceStore) Save(ctx context.Context, value aggregates.AttendanceDay, expectedRevision uint64) error {
	if ctx == nil || !value.ID().Valid() || !value.ParticipantID().Valid() || !value.Workday().Valid() || value.Revision() != expectedRevision+1 {
		return errors.New("valid attendance day revision is required")
	}
	now := s.now().UTC()
	ref := s.attendanceDayRef(value.ParticipantID(), value.Workday())
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snapshot, err := tx.Get(ref)
		if expectedRevision == 0 {
			if err == nil {
				return repositories.ErrConflict
			}
			if status.Code(err) != codes.NotFound {
				return err
			}
			return tx.Create(ref, storeAttendanceDay(value, now, now))
		}
		if err != nil {
			return err
		}
		var current attendanceDayDocument
		if err := snapshot.DataTo(&current); err != nil || current.SchemaVersion != attendanceDaySchemaVersion {
			return ErrRepositoryCorrupt
		}
		if current.Revision != int64(expectedRevision) {
			return repositories.ErrConcurrentUpdate
		}
		restored, err := restoreAttendanceDay(current)
		if err != nil || restored.ID() != value.ID() || restored.ParticipantID() != value.ParticipantID() || restored.Workday() != value.Workday() {
			return repositories.ErrConflict
		}
		if !validAttendanceDaySuccessor(restored, value) {
			return repositories.ErrConflict
		}
		return tx.Set(ref, storeAttendanceDay(value, current.CreatedAt, now))
	})
	return classifyAttendanceStoreError(ctx, err)
}

func validAttendanceDaySuccessor(current, next aggregates.AttendanceDay) bool {
	if next.Revision() != current.Revision()+1 {
		return false
	}
	currentIn, hasCurrentIn := current.ClockIn()
	nextIn, hasNextIn := next.ClockIn()
	_, hasCurrentOut := current.ClockOut()
	nextOut, hasNextOut := next.ClockOut()
	if !hasCurrentIn || !hasNextIn || !sameAttendancePunch(currentIn, nextIn) || hasCurrentOut || !hasNextOut {
		return false
	}
	return nextOut.RecordedAt().After(nextIn.RecordedAt())
}

func sameAttendancePunch(left, right interface {
	ID() valueobjects.PunchID
	Kind() valueobjects.PunchKind
	RecordedAt() time.Time
	Location() valueobjects.LocationFix
}) bool {
	return left.ID() == right.ID() && left.Kind() == right.Kind() && left.RecordedAt().Equal(right.RecordedAt()) && left.Location().Equal(right.Location())
}

var _ repositories.AttendanceDayRepository = (*AttendanceStore)(nil)
