package firestore

import (
	"context"
	"errors"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AttendanceStore) FindWorkplace(ctx context.Context, id valueobjects.WorkplaceID) (entities.Workplace, error) {
	if ctx == nil || !id.Valid() {
		return entities.Workplace{}, errors.New("valid workplace ID is required")
	}
	snapshot, err := s.client.Collection(attendanceWorkplacesCollection).Doc(attendanceDocumentID("workplace", string(id))).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return entities.Workplace{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.Workplace{}, attendanceStoreError(ctx, "read workplace")
	}
	var stored attendanceWorkplaceDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return entities.Workplace{}, ErrRepositoryCorrupt
	}
	return restoreAttendanceWorkplace(stored)
}

func (s *AttendanceStore) SaveWorkplace(ctx context.Context, workplace entities.Workplace) error {
	if ctx == nil || !workplace.ID().Valid() {
		return errors.New("valid workplace is required")
	}
	ref := s.client.Collection(attendanceWorkplacesCollection).Doc(attendanceDocumentID("workplace", string(workplace.ID())))
	now := s.now().UTC()
	return classifyAttendanceStoreError(ctx, s.upsertWithCreatedAt(ctx, ref, storeAttendanceWorkplace(workplace, now, now)))
}

func (s *AttendanceStore) FindShift(ctx context.Context, id valueobjects.ShiftInstanceID) (entities.ShiftInstance, error) {
	if ctx == nil || !id.Valid() {
		return entities.ShiftInstance{}, errors.New("valid shift ID is required")
	}
	snapshot, err := s.shiftRef(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return entities.ShiftInstance{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.ShiftInstance{}, attendanceStoreError(ctx, "read shift")
	}
	var stored attendanceShiftDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	return restoreAttendanceShift(stored)
}

func (s *AttendanceStore) SaveShift(ctx context.Context, shift entities.ShiftInstance) error {
	if ctx == nil || !shift.ID().Valid() {
		return errors.New("valid shift is required")
	}
	now := s.now().UTC()
	ref := s.shiftRef(shift.ID())
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return tx.Create(ref, storeAttendanceShift(shift, now, now))
		}
		if err != nil {
			return err
		}
		var stored attendanceShiftDocument
		if err := snapshot.DataTo(&stored); err != nil {
			return ErrRepositoryCorrupt
		}
		existing, err := restoreAttendanceShift(stored)
		if err != nil {
			return err
		}
		if !sameAttendanceShift(existing, shift) {
			return repositories.ErrConflict
		}
		return nil
	})
	return classifyAttendanceStoreError(ctx, err)
}

func (s *AttendanceStore) FindAssignment(ctx context.Context, id valueobjects.ShiftAssignmentID) (entities.ShiftAssignment, error) {
	if ctx == nil || !id.Valid() {
		return entities.ShiftAssignment{}, errors.New("valid assignment ID is required")
	}
	snapshot, err := s.assignmentRef(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return entities.ShiftAssignment{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.ShiftAssignment{}, attendanceStoreError(ctx, "read assignment")
	}
	var stored attendanceAssignmentDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	return restoreAttendanceAssignment(stored)
}

func (s *AttendanceStore) SaveAssignment(ctx context.Context, assignment entities.ShiftAssignment) error {
	if ctx == nil || !assignment.ID().Valid() {
		return errors.New("valid assignment is required")
	}
	now := s.now().UTC()
	ref := s.assignmentRef(assignment.ID())
	guardRef := s.assignmentGuardRef(assignment.EmployeeID(), assignment.ShiftInstanceID())
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		shiftSnapshot, err := tx.Get(s.shiftRef(assignment.ShiftInstanceID()))
		if status.Code(err) == codes.NotFound {
			return repositories.ErrNotFound
		}
		if err != nil {
			return err
		}
		var storedShift attendanceShiftDocument
		if err := shiftSnapshot.DataTo(&storedShift); err != nil {
			return ErrRepositoryCorrupt
		}
		shift, err := restoreAttendanceShift(storedShift)
		if err != nil {
			return err
		}
		if !shift.Active() {
			return repositories.ErrConflict
		}
		doc := storeAttendanceAssignment(assignment, shift, now, now)
		guardSnapshot, guardErr := tx.Get(guardRef)
		if guardErr == nil {
			var guard attendanceAssignmentGuardDocument
			if err := guardSnapshot.DataTo(&guard); err != nil || guard.SchemaVersion != attendanceSchemaVersion || guard.EmployeeID != string(assignment.EmployeeID()) || guard.ShiftInstanceID != string(assignment.ShiftInstanceID()) {
				return ErrRepositoryCorrupt
			}
			if guard.AssignmentID != string(assignment.ID()) {
				return repositories.ErrConflict
			}
		} else if status.Code(guardErr) != codes.NotFound {
			return guardErr
		}
		snapshot, assignmentErr := tx.Get(ref)
		if assignmentErr == nil {
			var existing attendanceAssignmentDocument
			if err := snapshot.DataTo(&existing); err != nil || existing.SchemaVersion != attendanceSchemaVersion || existing.ID != string(assignment.ID()) {
				return ErrRepositoryCorrupt
			}
			if existing.EmployeeID != string(assignment.EmployeeID()) || existing.ShiftInstanceID != string(assignment.ShiftInstanceID()) {
				return repositories.ErrConflict
			}
			doc.CreatedAt = existing.CreatedAt
		} else if status.Code(assignmentErr) != codes.NotFound {
			return assignmentErr
		}
		if (guardErr == nil) != (assignmentErr == nil) {
			return ErrRepositoryCorrupt
		}
		if status.Code(guardErr) == codes.NotFound {
			guard := attendanceAssignmentGuardDocument{SchemaVersion: attendanceSchemaVersion, AssignmentID: string(assignment.ID()), EmployeeID: string(assignment.EmployeeID()), ShiftInstanceID: string(assignment.ShiftInstanceID()), CreatedAt: now}
			if err := tx.Create(guardRef, guard); err != nil {
				return err
			}
		}
		if status.Code(assignmentErr) == codes.NotFound {
			return tx.Create(ref, doc)
		}
		return tx.Set(ref, doc)
	})
	return classifyAttendanceStoreError(ctx, err)
}

func (s *AttendanceStore) FindAssignmentForParticipant(ctx context.Context, participantID valueobjects.ParticipantID, shiftID valueobjects.ShiftInstanceID) (entities.ShiftAssignment, error) {
	if ctx == nil || !participantID.Valid() || !shiftID.Valid() {
		return entities.ShiftAssignment{}, errors.New("valid participant and shift IDs are required")
	}
	participant, err := s.FindByParticipantID(ctx, participantID)
	if err != nil {
		return entities.ShiftAssignment{}, err
	}
	if !participant.Active() {
		return entities.ShiftAssignment{}, repositories.ErrConflict
	}
	guardSnapshot, err := s.assignmentGuardRef(participant.EmployeeID(), shiftID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return entities.ShiftAssignment{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.ShiftAssignment{}, attendanceStoreError(ctx, "read assignment guard")
	}
	var guard attendanceAssignmentGuardDocument
	if err := guardSnapshot.DataTo(&guard); err != nil || guard.SchemaVersion != attendanceSchemaVersion || guard.EmployeeID != string(participant.EmployeeID()) || guard.ShiftInstanceID != string(shiftID) {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	assignmentID, err := valueobjects.NewShiftAssignmentID(guard.AssignmentID)
	if err != nil {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	assignment, err := s.FindAssignment(ctx, assignmentID)
	if err != nil {
		return entities.ShiftAssignment{}, err
	}
	if assignment.EmployeeID() != participant.EmployeeID() || assignment.ShiftInstanceID() != shiftID {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	return assignment, nil
}

func (s *AttendanceStore) ListOpenAssignments(ctx context.Context, participantID valueobjects.ParticipantID, at time.Time) ([]entities.ShiftAssignment, error) {
	participant, err := s.FindByParticipantID(ctx, participantID)
	if err != nil {
		return nil, err
	}
	if !participant.Active() {
		return nil, repositories.ErrConflict
	}
	query := s.client.Collection(attendanceAssignmentsCollection).
		Where("employee_id", "==", string(participant.EmployeeID())).
		Where("active", "==", true).
		Where("submission_opens_at", "<=", at.UTC()).
		Where("submission_closes_at", ">=", at.UTC()).
		Limit(maxParticipantAssignments + 1)
	iter := query.Documents(ctx)
	defer iter.Stop()
	var result []entities.ShiftAssignment
	scanned := 0
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, attendanceStoreError(ctx, "list assignments")
		}
		var stored attendanceAssignmentDocument
		if err := snap.DataTo(&stored); err != nil {
			return nil, ErrRepositoryCorrupt
		}
		assignment, err := restoreAttendanceAssignment(stored)
		if err != nil {
			return nil, err
		}
		scanned++
		if scanned > maxParticipantAssignments {
			return nil, repositories.ErrConflict
		}
		shift, err := s.FindShift(ctx, assignment.ShiftInstanceID())
		if err != nil {
			return nil, err
		}
		if shift.AcceptsSubmission(at) {
			result = append(result, assignment)
		}
	}
	return result, nil
}

func sameAttendanceShift(a, b entities.ShiftInstance) bool {
	return a.ID() == b.ID() &&
		a.WorkplaceID() == b.WorkplaceID() &&
		a.Label() == b.Label() &&
		a.Active() == b.Active() &&
		a.ClockInWindow().OpensAt().Equal(b.ClockInWindow().OpensAt()) &&
		a.ClockInWindow().ClosesAt().Equal(b.ClockInWindow().ClosesAt()) &&
		a.ClockOutWindow().OpensAt().Equal(b.ClockOutWindow().OpensAt()) &&
		a.ClockOutWindow().ClosesAt().Equal(b.ClockOutWindow().ClosesAt()) &&
		a.SubmissionOpensAt().Equal(b.SubmissionOpensAt()) &&
		a.SubmissionClosesAt().Equal(b.SubmissionClosesAt()) &&
		a.PolicyVersion() == b.PolicyVersion()
}

func (s *AttendanceStore) upsertWithCreatedAt(ctx context.Context, ref *sdk.DocumentRef, value any) error {
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snap, err := tx.Get(ref)
		if err == nil {
			switch d := value.(type) {
			case attendanceWorkplaceDocument:
				var current attendanceWorkplaceDocument
				if err := snap.DataTo(&current); err != nil {
					return ErrRepositoryCorrupt
				}
				d.CreatedAt = current.CreatedAt
				value = d
			case attendanceShiftDocument:
				var current attendanceShiftDocument
				if err := snap.DataTo(&current); err != nil {
					return ErrRepositoryCorrupt
				}
				d.CreatedAt = current.CreatedAt
				value = d
			case attendanceAssignmentDocument:
				var current attendanceAssignmentDocument
				if err := snap.DataTo(&current); err != nil {
					return ErrRepositoryCorrupt
				}
				d.CreatedAt = current.CreatedAt
				value = d
			}
		} else if status.Code(err) != codes.NotFound {
			return err
		}
		return tx.Set(ref, value)
	})
}
