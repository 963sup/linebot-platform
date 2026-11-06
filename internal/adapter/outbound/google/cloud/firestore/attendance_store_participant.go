package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		return entities.Participant{}, fmt.Errorf("%w: decode participant", ErrRepositoryCorrupt)
	}
	if !stored.Active && stored.ExpiresAt == nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	if stored.ExpiresAt != nil && retentionExpired(*stored.ExpiresAt, s.now().UTC()) {
		return entities.Participant{}, repositories.ErrNotFound
	}
	return restoreAttendanceParticipant(stored)
}

func (s *AttendanceStore) FindByParticipantID(ctx context.Context, id valueobjects.ParticipantID) (entities.Participant, error) {
	if ctx == nil || !id.Valid() {
		return entities.Participant{}, errors.New("valid participant ID is required")
	}
	snapshot, err := s.client.Collection(attendanceParticipantBindings).Doc(attendanceDocumentID("participant-binding", string(id))).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return entities.Participant{}, repositories.ErrNotFound
	}
	if err != nil {
		return entities.Participant{}, attendanceStoreError(ctx, "read participant binding")
	}
	var binding attendanceBindingDocument
	if err := snapshot.DataTo(&binding); err != nil || binding.SchemaVersion != attendanceSchemaVersion || binding.ParticipantID != string(id) {
		return entities.Participant{}, fmt.Errorf("%w: invalid participant binding", ErrRepositoryCorrupt)
	}
	if binding.ExpiresAt != nil && retentionExpired(*binding.ExpiresAt, s.now().UTC()) {
		return entities.Participant{}, repositories.ErrNotFound
	}
	employeeID, err := valueobjects.NewEmployeeID(binding.EmployeeID)
	if err != nil {
		return entities.Participant{}, fmt.Errorf("%w: invalid employee binding", ErrRepositoryCorrupt)
	}
	participant, err := s.FindByEmployeeID(ctx, employeeID)
	if err != nil {
		return entities.Participant{}, err
	}
	boundID, bound := participant.ParticipantID()
	if !bound || boundID != id {
		return entities.Participant{}, fmt.Errorf("%w: participant binding mismatch", ErrRepositoryCorrupt)
	}
	return participant, nil
}

func (s *AttendanceStore) SaveParticipant(ctx context.Context, participant entities.Participant) error {
	if ctx == nil || !participant.EmployeeID().Valid() {
		return errors.New("valid participant is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	now := s.now().UTC()
	ref := s.participantRef(participant.EmployeeID())
	return classifyAttendanceStoreError(ctx, s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		current, err := tx.Get(ref)
		createdAt := now
		if err == nil {
			var stored attendanceParticipantDocument
			if err := current.DataTo(&stored); err != nil || stored.SchemaVersion != attendanceSchemaVersion || stored.EmployeeID != string(participant.EmployeeID()) {
				return ErrRepositoryCorrupt
			}
			incomingParticipantID := ""
			if id, bound := participant.ParticipantID(); bound {
				incomingParticipantID = string(id)
			}
			if stored.ParticipantID != incomingParticipantID {
				return repositories.ErrConflict
			}
			createdAt = stored.CreatedAt
		} else if status.Code(err) != codes.NotFound {
			return err
		} else if _, bound := participant.ParticipantID(); bound {
			return repositories.ErrConflict
		}
		doc := storeAttendanceParticipant(participant, createdAt, now)
		var bindingRef *sdk.DocumentRef
		if participantID, bound := participant.ParticipantID(); bound && doc.ExpiresAt != nil {
			bindingRef = s.client.Collection(attendanceParticipantBindings).Doc(attendanceDocumentID("participant-binding", string(participantID)))
			bindingSnapshot, err := tx.Get(bindingRef)
			if status.Code(err) == codes.NotFound {
				return ErrRepositoryCorrupt
			}
			if err != nil {
				return err
			}
			var binding attendanceBindingDocument
			if err := bindingSnapshot.DataTo(&binding); err != nil || binding.SchemaVersion != attendanceSchemaVersion || binding.ParticipantID != string(participantID) || binding.EmployeeID != string(participant.EmployeeID()) {
				return ErrRepositoryCorrupt
			}
		}
		if err := tx.Set(ref, doc); err != nil {
			return err
		}
		if bindingRef != nil {
			if err := tx.Update(bindingRef, []sdk.Update{{Path: "expires_at", Value: *doc.ExpiresAt}}); err != nil {
				return err
			}
		}
		return nil
	}))
}

func (s *AttendanceStore) Invite(ctx context.Context, participant entities.Participant, invitation entities.EnrollmentInvitation) error {
	if ctx == nil || !invitation.EmployeeID().Valid() || !invitation.CodeHash().Valid() || participant.EmployeeID() != invitation.EmployeeID() || !participant.Active() {
		return errors.New("valid enrollment invitation is required")
	}
	now := s.now().UTC()
	if !invitation.ExpiresAt().After(now) || invitation.ExpiresAt().After(now.Add(24*time.Hour)) {
		return errors.New("enrollment invitation expiry must be within 24 hours")
	}
	ref := s.enrollmentRef(invitation.CodeHash())
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		participantRef := s.participantRef(invitation.EmployeeID())
		participantSnapshot, participantErr := tx.Get(participantRef)
		if participantErr == nil {
			var stored attendanceParticipantDocument
			if err := participantSnapshot.DataTo(&stored); err != nil {
				return ErrRepositoryCorrupt
			}
			existing, err := restoreAttendanceParticipant(stored)
			if err != nil || existing.EmployeeCode() != participant.EmployeeCode() || existing.DisplayName() != participant.DisplayName() {
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
		return tx.Create(ref, attendanceEnrollmentDocument{SchemaVersion: attendanceSchemaVersion, CodeHash: string(invitation.CodeHash()), EmployeeID: string(invitation.EmployeeID()), ExpiresAt: invitation.ExpiresAt(), CreatedAt: now})
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
		tokenSnap, err := tx.Get(tokenRef)
		if status.Code(err) == codes.NotFound {
			return repositories.ErrNotFound
		}
		if err != nil {
			return err
		}
		var token attendanceEnrollmentDocument
		if err := tokenSnap.DataTo(&token); err != nil || token.SchemaVersion != attendanceSchemaVersion || token.CodeHash != string(codeHash) {
			return ErrRepositoryCorrupt
		}
		if token.ConsumedAt != nil {
			if token.ParticipantID != string(participantID) {
				return repositories.ErrConflict
			}
			employeeID, err := valueobjects.NewEmployeeID(token.EmployeeID)
			if err != nil {
				return ErrRepositoryCorrupt
			}
			participantSnap, err := tx.Get(s.participantRef(employeeID))
			if err != nil {
				return err
			}
			var participantDoc attendanceParticipantDocument
			if err := participantSnap.DataTo(&participantDoc); err != nil {
				return ErrRepositoryCorrupt
			}
			result, err = restoreAttendanceParticipant(participantDoc)
			if err != nil {
				return err
			}
			boundParticipantID, bound := result.ParticipantID()
			if !bound || boundParticipantID != participantID {
				return ErrRepositoryCorrupt
			}
			bindingRef := s.client.Collection(attendanceParticipantBindings).Doc(attendanceDocumentID("participant-binding", string(participantID)))
			bindingSnapshot, bindingErr := tx.Get(bindingRef)
			if status.Code(bindingErr) == codes.NotFound {
				return ErrRepositoryCorrupt
			}
			if bindingErr != nil {
				return bindingErr
			}
			var binding attendanceBindingDocument
			if err := bindingSnapshot.DataTo(&binding); err != nil || binding.SchemaVersion != attendanceSchemaVersion || binding.ParticipantID != string(participantID) || binding.EmployeeID != string(employeeID) {
				return ErrRepositoryCorrupt
			}
			outcome = valueobjects.EnrollmentAlreadyBound
			return nil
		}
		if retentionExpired(token.ExpiresAt, serverTime) {
			return repositories.ErrConflict
		}
		employeeID, err := valueobjects.NewEmployeeID(token.EmployeeID)
		if err != nil {
			return ErrRepositoryCorrupt
		}
		participantRef := s.participantRef(employeeID)
		participantSnap, err := tx.Get(participantRef)
		if err != nil {
			return err
		}
		var participantDoc attendanceParticipantDocument
		if err := participantSnap.DataTo(&participantDoc); err != nil {
			return ErrRepositoryCorrupt
		}
		participant, err := restoreAttendanceParticipant(participantDoc)
		if err != nil {
			return err
		}
		if !participant.Active() {
			return repositories.ErrConflict
		}
		if existing, ok := participant.ParticipantID(); ok && existing != participantID {
			return repositories.ErrConflict
		}
		bindingRef := s.client.Collection(attendanceParticipantBindings).Doc(attendanceDocumentID("participant-binding", string(participantID)))
		bindingSnap, bindingErr := tx.Get(bindingRef)
		if bindingErr == nil {
			var binding attendanceBindingDocument
			if err := bindingSnap.DataTo(&binding); err != nil || binding.EmployeeID != string(employeeID) || binding.ParticipantID != string(participantID) {
				return repositories.ErrConflict
			}
		} else if status.Code(bindingErr) != codes.NotFound {
			return bindingErr
		}
		bindOutcome, err := participant.Bind(participantID)
		if err != nil {
			return repositories.ErrConflict
		}
		outcome = bindOutcome
		participantDoc = storeAttendanceParticipant(participant, participantDoc.CreatedAt, serverTime)
		if err := tx.Set(participantRef, participantDoc); err != nil {
			return err
		}
		if status.Code(bindingErr) == codes.NotFound {
			if err := tx.Create(bindingRef, attendanceBindingDocument{SchemaVersion: attendanceSchemaVersion, ParticipantID: string(participantID), EmployeeID: string(employeeID), CreatedAt: serverTime}); err != nil {
				return err
			}
		}
		token.ConsumedAt, token.ParticipantID = &serverTime, string(participantID)
		if err := tx.Set(tokenRef, token); err != nil {
			return err
		}
		result = participant
		return nil
	})
	return result, outcome, classifyAttendanceStoreError(ctx, err)
}
