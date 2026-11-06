package firestore

import (
	"context"
	"errors"
	"sort"
	"time"

	sdk "cloud.google.com/go/firestore"
	attendanceapp "github.com/963sup/linebot-platform/internal/application/usecases/attendance"
	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ResolvePunchContext returns one coherent current view. All documents are
// read in the same read-only Firestore transaction; a missing attendance
// document is represented only by a nil snapshot Attendance.
func (s *AttendanceStore) ResolvePunchContext(ctx context.Context, participantID valueobjects.ParticipantID, shiftID valueobjects.ShiftInstanceID, at time.Time) (attendanceapp.CurrentAttendanceSnapshot, error) {
	if ctx == nil || !participantID.Valid() || !shiftID.Valid() || at.IsZero() {
		return attendanceapp.CurrentAttendanceSnapshot{}, errors.New("valid current attendance identity and time are required")
	}
	at = at.UTC()
	var result attendanceapp.CurrentAttendanceSnapshot
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *sdk.Transaction) error {
		participant, err := s.readCurrentParticipant(tx, participantID, at)
		if err != nil {
			return err
		}
		guardSnapshot, err := tx.Get(s.assignmentGuardRef(participant.EmployeeID(), shiftID))
		if status.Code(err) == codes.NotFound {
			return repositories.ErrNotFound
		}
		if err != nil {
			return err
		}
		var guard attendanceAssignmentGuardDocument
		if err := guardSnapshot.DataTo(&guard); err != nil || guard.SchemaVersion != attendanceSchemaVersion || guard.EmployeeID != string(participant.EmployeeID()) || guard.ShiftInstanceID != string(shiftID) {
			return ErrRepositoryCorrupt
		}
		assignmentID, err := valueobjects.NewShiftAssignmentID(guard.AssignmentID)
		if err != nil {
			return ErrRepositoryCorrupt
		}
		snapshots, err := tx.GetAll([]*sdk.DocumentRef{
			s.assignmentRef(assignmentID),
			s.shiftRef(shiftID),
			s.attendanceRef(participantID, assignmentID),
		})
		if err != nil {
			return err
		}
		if !snapshots[0].Exists() || !snapshots[1].Exists() {
			return ErrRepositoryCorrupt
		}
		assignment, err := decodeCurrentAssignment(snapshots[0])
		if err != nil {
			return err
		}
		shift, err := decodeCurrentShift(snapshots[1])
		if err != nil {
			return err
		}
		workplaceSnapshot, err := tx.Get(s.client.Collection(attendanceWorkplacesCollection).Doc(attendanceDocumentID("workplace", string(shift.WorkplaceID()))))
		if status.Code(err) == codes.NotFound {
			return ErrRepositoryCorrupt
		}
		if err != nil {
			return err
		}
		workplace, err := decodeCurrentWorkplace(workplaceSnapshot)
		if err != nil {
			return err
		}
		attendance, err := decodeOptionalCurrentAttendance(snapshots[2], at)
		if err != nil {
			return err
		}
		candidate := attendanceapp.CurrentAttendanceSnapshot{
			Participant: participant,
			Assignment:  assignment,
			Shift:       shift,
			Workplace:   workplace,
			Attendance:  attendance,
		}
		if !currentAttendanceSnapshotCoherent(candidate, participantID, shiftID) {
			return ErrRepositoryCorrupt
		}
		result = candidate
		return nil
	}, sdk.ReadOnly)
	if err != nil {
		return attendanceapp.CurrentAttendanceSnapshot{}, classifyAttendanceStoreError(ctx, err)
	}
	return result, nil
}

// ListCurrentOptions resolves all current assignment options with a constant
// number of Firestore round trips. A corrupt or dangling row fails the entire
// view rather than returning a partial set.
func (s *AttendanceStore) ListCurrentOptions(ctx context.Context, participantID valueobjects.ParticipantID, at time.Time) ([]attendanceapp.CurrentAttendanceSnapshot, error) {
	if ctx == nil || !participantID.Valid() || at.IsZero() {
		return nil, errors.New("valid current attendance participant and time are required")
	}
	at = at.UTC()
	var result []attendanceapp.CurrentAttendanceSnapshot
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *sdk.Transaction) error {
		participant, err := s.readCurrentParticipant(tx, participantID, at)
		if err != nil {
			return err
		}
		if !participant.Active() {
			return repositories.ErrConflict
		}
		query := s.client.Collection(attendanceAssignmentsCollection).
			Where("employee_id", "==", string(participant.EmployeeID())).
			Where("active", "==", true).
			Where("submission_opens_at", "<=", at).
			Where("submission_closes_at", ">=", at).
			Limit(maxParticipantAssignments + 1)
		iter := tx.Documents(query)
		defer iter.Stop()
		assignments := make([]entities.ShiftAssignment, 0)
		for {
			snapshot, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			assignment, err := decodeCurrentAssignment(snapshot)
			if err != nil {
				return err
			}
			if assignment.EmployeeID() != participant.EmployeeID() {
				return ErrRepositoryCorrupt
			}
			assignments = append(assignments, assignment)
			if len(assignments) > maxParticipantAssignments {
				return repositories.ErrConflict
			}
		}
		if len(assignments) == 0 {
			result = []attendanceapp.CurrentAttendanceSnapshot{}
			return nil
		}

		shiftRefs := make([]*sdk.DocumentRef, len(assignments))
		attendanceRefs := make([]*sdk.DocumentRef, len(assignments))
		for i, assignment := range assignments {
			shiftRefs[i] = s.shiftRef(assignment.ShiftInstanceID())
			attendanceRefs[i] = s.attendanceRef(participantID, assignment.ID())
		}
		shiftSnapshots, err := tx.GetAll(shiftRefs)
		if err != nil {
			return err
		}
		shifts := make([]entities.ShiftInstance, len(assignments))
		workplaceRefs := make([]*sdk.DocumentRef, len(assignments))
		for i, snapshot := range shiftSnapshots {
			if !snapshot.Exists() {
				return ErrRepositoryCorrupt
			}
			shift, err := decodeCurrentShift(snapshot)
			if err != nil || shift.ID() != assignments[i].ShiftInstanceID() || !shift.Active() || !shift.AcceptsSubmission(at) {
				return ErrRepositoryCorrupt
			}
			shifts[i] = shift
			workplaceRefs[i] = s.client.Collection(attendanceWorkplacesCollection).Doc(attendanceDocumentID("workplace", string(shift.WorkplaceID())))
		}
		workplaceSnapshots, err := tx.GetAll(workplaceRefs)
		if err != nil {
			return err
		}
		attendanceSnapshots, err := tx.GetAll(attendanceRefs)
		if err != nil {
			return err
		}
		options := make([]attendanceapp.CurrentAttendanceSnapshot, len(assignments))
		for i := range assignments {
			if !workplaceSnapshots[i].Exists() {
				return ErrRepositoryCorrupt
			}
			workplace, err := decodeCurrentWorkplace(workplaceSnapshots[i])
			if err != nil || !workplace.Active() {
				return ErrRepositoryCorrupt
			}
			attendance, err := decodeOptionalCurrentAttendance(attendanceSnapshots[i], at)
			if err != nil {
				return err
			}
			candidate := attendanceapp.CurrentAttendanceSnapshot{
				Participant: participant,
				Assignment:  assignments[i],
				Shift:       shifts[i],
				Workplace:   workplace,
				Attendance:  attendance,
			}
			if !currentAttendanceSnapshotCoherent(candidate, participantID, shifts[i].ID()) {
				return ErrRepositoryCorrupt
			}
			options[i] = candidate
		}
		sort.Slice(options, func(i, j int) bool {
			if options[i].Shift.StartsAt().Equal(options[j].Shift.StartsAt()) {
				return options[i].Shift.ID() < options[j].Shift.ID()
			}
			return options[i].Shift.StartsAt().Before(options[j].Shift.StartsAt())
		})
		result = options
		return nil
	}, sdk.ReadOnly)
	if err != nil {
		return nil, classifyAttendanceStoreError(ctx, err)
	}
	return result, nil
}

func (s *AttendanceStore) readCurrentParticipant(tx *sdk.Transaction, participantID valueobjects.ParticipantID, at time.Time) (entities.Participant, error) {
	bindingSnapshot, err := tx.Get(s.client.Collection(attendanceParticipantBindings).Doc(attendanceDocumentID("participant-binding", string(participantID))))
	if status.Code(err) == codes.NotFound {
		return entities.Participant{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.Participant{}, err
	}
	var binding attendanceBindingDocument
	if err := bindingSnapshot.DataTo(&binding); err != nil || binding.SchemaVersion != attendanceSchemaVersion || binding.ParticipantID != string(participantID) || binding.EmployeeID == "" {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	if binding.ExpiresAt != nil && retentionExpired(*binding.ExpiresAt, at) {
		return entities.Participant{}, repositories.ErrNotFound
	}
	employeeID, err := valueobjects.NewEmployeeID(binding.EmployeeID)
	if err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	participantSnapshot, err := tx.Get(s.participantRef(employeeID))
	if status.Code(err) == codes.NotFound {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	if err != nil {
		return entities.Participant{}, err
	}
	var stored attendanceParticipantDocument
	if err := participantSnapshot.DataTo(&stored); err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	participant, err := restoreAttendanceParticipant(stored)
	if err != nil {
		return entities.Participant{}, err
	}
	boundID, bound := participant.ParticipantID()
	if !bound || boundID != participantID || participant.EmployeeID() != employeeID {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	return participant, nil
}

func decodeCurrentAssignment(snapshot *sdk.DocumentSnapshot) (entities.ShiftAssignment, error) {
	var stored attendanceAssignmentDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	return restoreAttendanceAssignment(stored)
}

func decodeCurrentShift(snapshot *sdk.DocumentSnapshot) (entities.ShiftInstance, error) {
	var stored attendanceShiftDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	return restoreAttendanceShift(stored)
}

func decodeCurrentWorkplace(snapshot *sdk.DocumentSnapshot) (entities.Workplace, error) {
	var stored attendanceWorkplaceDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return entities.Workplace{}, ErrRepositoryCorrupt
	}
	return restoreAttendanceWorkplace(stored)
}

func decodeOptionalCurrentAttendance(snapshot *sdk.DocumentSnapshot, at time.Time) (*aggregates.Attendance, error) {
	if !snapshot.Exists() {
		return nil, nil
	}
	var stored attendanceRecordDocument
	if err := snapshot.DataTo(&stored); err != nil || stored.ExpiresAt.IsZero() {
		return nil, ErrRepositoryCorrupt
	}
	if retentionExpired(stored.ExpiresAt, at) {
		return nil, nil
	}
	record, err := restoreAttendanceRecord(stored)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func currentAttendanceSnapshotCoherent(snapshot attendanceapp.CurrentAttendanceSnapshot, participantID valueobjects.ParticipantID, shiftID valueobjects.ShiftInstanceID) bool {
	boundID, bound := snapshot.Participant.ParticipantID()
	if !bound || boundID != participantID || !snapshot.Assignment.ID().Valid() || snapshot.Assignment.EmployeeID() != snapshot.Participant.EmployeeID() || snapshot.Assignment.ShiftInstanceID() != shiftID || snapshot.Shift.ID() != shiftID || snapshot.Shift.WorkplaceID() != snapshot.Workplace.ID() {
		return false
	}
	if snapshot.Attendance == nil {
		return true
	}
	storedParticipantID, storedBound := snapshot.Attendance.Participant().ParticipantID()
	return storedBound && storedParticipantID == participantID &&
		snapshot.Attendance.Assignment().ID() == snapshot.Assignment.ID() &&
		snapshot.Attendance.Shift().ID() == snapshot.Shift.ID() &&
		snapshot.Attendance.Workplace().ID() == snapshot.Workplace.ID()
}

var _ attendanceapp.CurrentAttendanceReader = (*AttendanceStore)(nil)
