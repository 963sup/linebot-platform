package firestore

import (
	"context"
	"errors"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AttendanceStore) participantRef(id valueobjects.EmployeeID) *sdk.DocumentRef {
	return s.client.Collection(attendanceParticipantsCollection).Doc(attendanceDocumentID("employee", string(id)))
}

func (s *AttendanceStore) enrollmentRef(hash valueobjects.EnrollmentCodeHash) *sdk.DocumentRef {
	return s.client.Collection(attendanceEnrollmentTokensCollection).Doc(attendanceDocumentID("enrollment", string(hash)))
}

func (s *AttendanceStore) bindingRef(id valueobjects.ParticipantID) *sdk.DocumentRef {
	return s.client.Collection(attendanceParticipantBindings).Doc(attendanceDocumentID("participant-binding", string(id)))
}

func (s *AttendanceStore) FindByEmployeeID(ctx context.Context, id valueobjects.EmployeeID) (entities.Participant, error) {
	if ctx == nil || !id.Valid() {
		return entities.Participant{}, errors.New("valid employee ID is required")
	}
	snapshot, err := s.participantRef(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return entities.Participant{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.Participant{}, attendanceStoreError(ctx, "read participant")
	}
	var stored attendanceParticipantDocument
	if err := snapshot.DataTo(&stored); err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	return restoreAttendanceParticipant(stored)
}

func (s *AttendanceStore) FindByParticipantID(ctx context.Context, id valueobjects.ParticipantID) (entities.Participant, error) {
	if ctx == nil || !id.Valid() {
		return entities.Participant{}, errors.New("valid participant ID is required")
	}
	bindingSnapshot, err := s.bindingRef(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return entities.Participant{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.Participant{}, attendanceStoreError(ctx, "read participant binding")
	}
	var binding attendanceBindingDocument
	if err := bindingSnapshot.DataTo(&binding); err != nil || binding.SchemaVersion != attendanceParticipantSchemaVersion || binding.ParticipantID != string(id) {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	employeeID, err := valueobjects.NewEmployeeID(binding.EmployeeID)
	if err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	participant, err := s.FindByEmployeeID(ctx, employeeID)
	if err != nil {
		return entities.Participant{}, err
	}
	boundID, bound := participant.ParticipantID()
	if !bound || boundID != id {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	return participant, nil
}

func (s *AttendanceStore) SaveParticipant(ctx context.Context, participant entities.Participant) error {
	if ctx == nil || !participant.EmployeeID().Valid() {
		return errors.New("valid participant is required")
	}
	now := s.now().UTC()
	ref := s.participantRef(participant.EmployeeID())
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		current, err := tx.Get(ref)
		createdAt := now
		if err == nil {
			var stored attendanceParticipantDocument
			if err := current.DataTo(&stored); err != nil || stored.SchemaVersion != attendanceParticipantSchemaVersion || stored.EmployeeID != string(participant.EmployeeID()) {
				return ErrRepositoryCorrupt
			}
			incomingID := ""
			if id, bound := participant.ParticipantID(); bound {
				incomingID = string(id)
			}
			if stored.ParticipantID != incomingID {
				return repositories.ErrConflict
			}
			createdAt = stored.CreatedAt
		} else if status.Code(err) != codes.NotFound {
			return err
		} else if _, bound := participant.ParticipantID(); bound {
			return repositories.ErrConflict
		}
		return tx.Set(ref, storeAttendanceParticipant(participant, createdAt, now))
	})
	return classifyAttendanceStoreError(ctx, err)
}

func (s *AttendanceStore) Invite(ctx context.Context, participant entities.Participant, invitation entities.EnrollmentInvitation) error {
	if ctx == nil || !invitation.EmployeeID().Valid() || !invitation.CodeHash().Valid() || participant.EmployeeID() != invitation.EmployeeID() || !participant.Active() {
		return errors.New("valid enrollment invitation is required")
	}
	now := s.now().UTC()
	if !invitation.ExpiresAt().After(now) || invitation.ExpiresAt().After(now.Add(24*time.Hour)) {
		return errors.New("enrollment invitation expiry must be within 24 hours")
	}
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		participantRef := s.participantRef(participant.EmployeeID())
		participantSnapshot, participantErr := tx.Get(participantRef)
		if participantErr == nil {
			var stored attendanceParticipantDocument
			if err := participantSnapshot.DataTo(&stored); err != nil {
				return ErrRepositoryCorrupt
			}
			existing, err := restoreAttendanceParticipant(stored)
			if err != nil || existing.EmployeeCode() != participant.EmployeeCode() || existing.DisplayName() != participant.DisplayName() || !existing.Active() {
				return repositories.ErrConflict
			}
			if _, bound := existing.ParticipantID(); bound {
				return repositories.ErrConflict
			}
		} else if status.Code(participantErr) == codes.NotFound {
			if err := tx.Create(participantRef, storeAttendanceParticipant(participant, now, now)); err != nil {
				return err
			}
		} else {
			return participantErr
		}
		return tx.Create(s.enrollmentRef(invitation.CodeHash()), attendanceEnrollmentDocument{
			SchemaVersion: attendanceParticipantSchemaVersion, CodeHash: string(invitation.CodeHash()), EmployeeID: string(invitation.EmployeeID()),
			ExpiresAt: invitation.ExpiresAt(), CreatedAt: now,
		})
	})
	return classifyAttendanceStoreError(ctx, err)
}

func (s *AttendanceStore) ConsumeEnrollment(ctx context.Context, codeHash valueobjects.EnrollmentCodeHash, participantID valueobjects.ParticipantID, serverTime time.Time) (entities.Participant, valueobjects.EnrollmentOutcome, error) {
	if ctx == nil || !codeHash.Valid() || !participantID.Valid() || serverTime.IsZero() {
		return entities.Participant{}, "", errors.New("valid enrollment consumption is required")
	}
	serverTime = serverTime.UTC()
	var result entities.Participant
	var outcome valueobjects.EnrollmentOutcome
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		tokenRef := s.enrollmentRef(codeHash)
		tokenSnapshot, err := tx.Get(tokenRef)
		if status.Code(err) == codes.NotFound {
			return repositories.ErrNotFound
		}
		if err != nil {
			return err
		}
		var token attendanceEnrollmentDocument
		if err := tokenSnapshot.DataTo(&token); err != nil || token.SchemaVersion != attendanceParticipantSchemaVersion || token.CodeHash != string(codeHash) {
			return ErrRepositoryCorrupt
		}
		if token.ConsumedAt != nil {
			if token.ParticipantID != string(participantID) {
				return repositories.ErrConflict
			}
			return s.restoreConsumedEnrollment(tx, token, participantID, &result, &outcome)
		}
		if retentionExpired(token.ExpiresAt, serverTime) {
			return repositories.ErrConflict
		}
		employeeID, err := valueobjects.NewEmployeeID(token.EmployeeID)
		if err != nil {
			return ErrRepositoryCorrupt
		}
		participantRef := s.participantRef(employeeID)
		participantSnapshot, err := tx.Get(participantRef)
		if err != nil {
			return err
		}
		var participantDoc attendanceParticipantDocument
		if err := participantSnapshot.DataTo(&participantDoc); err != nil {
			return ErrRepositoryCorrupt
		}
		participant, err := restoreAttendanceParticipant(participantDoc)
		if err != nil || !participant.Active() {
			return repositories.ErrConflict
		}
		bindingRef := s.bindingRef(participantID)
		bindingSnapshot, bindingErr := tx.Get(bindingRef)
		if bindingErr == nil {
			var binding attendanceBindingDocument
			if err := bindingSnapshot.DataTo(&binding); err != nil || binding.SchemaVersion != attendanceParticipantSchemaVersion || binding.EmployeeID != string(employeeID) || binding.ParticipantID != string(participantID) {
				return repositories.ErrConflict
			}
		} else if status.Code(bindingErr) != codes.NotFound {
			return bindingErr
		}
		bindOutcome, err := participant.Bind(participantID)
		if err != nil {
			return repositories.ErrConflict
		}
		if err := tx.Set(participantRef, storeAttendanceParticipant(participant, participantDoc.CreatedAt, serverTime)); err != nil {
			return err
		}
		if status.Code(bindingErr) == codes.NotFound {
			if err := tx.Create(bindingRef, attendanceBindingDocument{SchemaVersion: attendanceParticipantSchemaVersion, ParticipantID: string(participantID), EmployeeID: string(employeeID), CreatedAt: serverTime}); err != nil {
				return err
			}
		}
		token.ConsumedAt, token.ParticipantID = &serverTime, string(participantID)
		if err := tx.Set(tokenRef, token); err != nil {
			return err
		}
		result, outcome = participant, bindOutcome
		return nil
	})
	return result, outcome, classifyAttendanceStoreError(ctx, err)
}

func (s *AttendanceStore) restoreConsumedEnrollment(tx *sdk.Transaction, token attendanceEnrollmentDocument, participantID valueobjects.ParticipantID, result *entities.Participant, outcome *valueobjects.EnrollmentOutcome) error {
	employeeID, err := valueobjects.NewEmployeeID(token.EmployeeID)
	if err != nil {
		return ErrRepositoryCorrupt
	}
	participantSnapshot, err := tx.Get(s.participantRef(employeeID))
	if err != nil {
		return err
	}
	var stored attendanceParticipantDocument
	if err := participantSnapshot.DataTo(&stored); err != nil {
		return ErrRepositoryCorrupt
	}
	participant, err := restoreAttendanceParticipant(stored)
	if err != nil {
		return err
	}
	boundID, bound := participant.ParticipantID()
	if !bound || boundID != participantID {
		return ErrRepositoryCorrupt
	}
	bindingSnapshot, err := tx.Get(s.bindingRef(participantID))
	if err != nil {
		return err
	}
	var binding attendanceBindingDocument
	if err := bindingSnapshot.DataTo(&binding); err != nil || binding.SchemaVersion != attendanceParticipantSchemaVersion || binding.EmployeeID != string(employeeID) || binding.ParticipantID != string(participantID) {
		return ErrRepositoryCorrupt
	}
	*result, *outcome = participant, valueobjects.EnrollmentAlreadyBound
	return nil
}

var _ repositories.ParticipantRepository = (*AttendanceStore)(nil)
var _ repositories.EnrollmentRepository = (*AttendanceStore)(nil)
