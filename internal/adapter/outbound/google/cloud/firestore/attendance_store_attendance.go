package firestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AttendanceStore) Find(ctx context.Context, participantID valueobjects.ParticipantID, assignmentID valueobjects.ShiftAssignmentID) (aggregates.Attendance, error) {
	if ctx == nil || !participantID.Valid() || !assignmentID.Valid() {
		return aggregates.Attendance{}, errors.New("valid attendance identity is required")
	}
	ref := s.attendanceRef(participantID, assignmentID)
	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return aggregates.Attendance{}, repositories.ErrNotFound
	}
	if err != nil {
		return aggregates.Attendance{}, attendanceStoreError(ctx, "read attendance")
	}
	var stored attendanceRecordDocument
	if err := snap.DataTo(&stored); err != nil {
		return aggregates.Attendance{}, ErrRepositoryCorrupt
	}
	if stored.ExpiresAt.IsZero() {
		return aggregates.Attendance{}, ErrRepositoryCorrupt
	}
	if retentionExpired(stored.ExpiresAt, s.now().UTC()) {
		return aggregates.Attendance{}, repositories.ErrNotFound
	}
	return restoreAttendanceRecord(stored)
}

func (s *AttendanceStore) SavePunch(ctx context.Context, attendance aggregates.Attendance, expectedRevision uint64, punchID valueobjects.PunchID, evidence valueobjects.LocationEvidence, evidenceExpiresAt time.Time) error {
	if !punchID.Valid() || evidenceExpiresAt.IsZero() {
		return errors.New("valid attendance punch evidence is required")
	}
	return s.saveAttendance(ctx, attendance, expectedRevision, &punchEvidence{punchID: punchID, evidence: evidence, expiresAt: evidenceExpiresAt.UTC()})
}

func (s *AttendanceStore) SaveReview(ctx context.Context, attendance aggregates.Attendance, expectedRevision uint64) error {
	return s.saveAttendance(ctx, attendance, expectedRevision, nil)
}

type punchEvidence struct {
	punchID   valueobjects.PunchID
	evidence  valueobjects.LocationEvidence
	expiresAt time.Time
}

func (s *AttendanceStore) saveAttendance(ctx context.Context, attendance aggregates.Attendance, expectedRevision uint64, evidence *punchEvidence) error {
	participantID, bound := attendance.Participant().ParticipantID()
	if ctx == nil || !bound || attendance.Revision() != expectedRevision+1 {
		return errors.New("valid attendance revision is required")
	}
	now := s.now().UTC()
	incoming, err := storeAttendanceRecord(attendance, now)
	if err != nil {
		return err
	}
	var evidenceDoc *attendanceEvidenceDocument
	var evidenceRef *sdk.DocumentRef
	if evidence != nil {
		punch, ok := newPunchForRevision(attendance, expectedRevision)
		if !ok || punch.ID() != evidence.punchID || !evidence.expiresAt.Equal(punch.RecordedAt().Add(locationEvidenceRetention)) {
			return errors.New("attendance evidence has no new punch")
		}
		if expectedRevision == 0 {
			clockIn, hasClockIn := attendance.ClockIn()
			_, hasClockOut := attendance.ClockOut()
			if !hasClockIn || hasClockOut || len(clockIn.ReviewHistory()) != 0 {
				return errors.New("first attendance revision must contain only clock in")
			}
		}
		expectedRef, err := valueobjects.NewEvidenceRef(string(punch.ID()))
		if err != nil || punch.Location().EvidenceRef() != expectedRef {
			return errors.New("attendance evidence reference does not belong to punch")
		}
		expectedAssessment, err := attendance.Workplace().AssessLocation(evidence.evidence, expectedRef)
		if err != nil || !expectedAssessment.Equal(punch.Location()) {
			return errors.New("attendance evidence does not match location assessment")
		}
		plaintext, err := json.Marshal(struct{ Latitude, Longitude, AccuracyMeters float64 }{evidence.evidence.Latitude(), evidence.evidence.Longitude(), evidence.evidence.AccuracyMeters()})
		if err != nil {
			return errors.New("encode attendance evidence")
		}
		keyID, ciphertext, err := s.keyring.Seal(string(punch.ID()), plaintext)
		if err != nil {
			return err
		}
		evidenceRef = s.client.Collection(attendanceEvidenceCollection).Doc(attendanceDocumentID("location-evidence", string(punch.ID())))
		evidenceDoc = &attendanceEvidenceDocument{SchemaVersion: attendanceSchemaVersion, PunchID: string(punch.ID()), KeyID: keyID, Ciphertext: ciphertext, CreatedAt: punch.RecordedAt(), ExpiresAt: evidence.expiresAt}
	}
	ref := s.attendanceRef(participantID, attendance.Assignment().ID())
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		currentSnap, getErr := tx.Get(ref)
		currentActions := map[string]int{}
		if expectedRevision == 0 {
			if getErr == nil {
				return repositories.ErrConflict
			}
			if status.Code(getErr) != codes.NotFound {
				return getErr
			}
			incoming.CreatedAt = now
			incoming.ExpiresAt = attendanceExpiry(retentionBase(incoming))
		} else {
			if getErr != nil {
				return getErr
			}
			var current attendanceRecordDocument
			if err := currentSnap.DataTo(&current); err != nil || current.SchemaVersion != attendanceSchemaVersion {
				return ErrRepositoryCorrupt
			}
			if current.Revision != int64(expectedRevision) {
				return repositories.ErrConcurrentUpdate
			}
			if err := validateAttendanceUpdate(current, incoming, evidence != nil); err != nil {
				return err
			}
			incoming.CreatedAt = current.CreatedAt
			incoming.ExpiresAt = current.ExpiresAt
			currentActions = reviewActionCounts(current)
		}
		if evidenceDoc != nil {
			if err := tx.Create(evidenceRef, *evidenceDoc); err != nil {
				return err
			}
		}
		if expectedRevision == 0 {
			if err := tx.Create(ref, incoming); err != nil {
				return err
			}
		} else if err := tx.Set(ref, incoming); err != nil {
			return err
		}
		return s.createReviewActions(tx, attendanceDocumentID("attendance", string(participantID), string(attendance.Assignment().ID())), incoming, currentActions)
	})
	return classifyAttendanceStoreError(ctx, err)
}

func validateAttendanceUpdate(current, incoming attendanceRecordDocument, savingPunch bool) error {
	if _, err := restoreAttendanceRecord(current); err != nil {
		return ErrRepositoryCorrupt
	}
	if !sameStoredParticipant(current.Participant, incoming.Participant) ||
		!sameStoredWorkplace(current.Workplace, incoming.Workplace) ||
		!sameStoredShift(current.Shift, incoming.Shift) ||
		!sameStoredAssignment(current.Assignment, incoming.Assignment) {
		return repositories.ErrConflict
	}
	if savingPunch {
		if current.ClockOut != nil || incoming.ClockOut == nil || len(incoming.ClockOut.History) != 0 || !sameStoredPunch(current.ClockIn, incoming.ClockIn) {
			return repositories.ErrConflict
		}
		return nil
	}
	clockInActions, err := validateReviewAppend(current.ClockIn, incoming.ClockIn)
	if err != nil {
		return err
	}
	clockOutActions, err := validateReviewAppend(current.ClockOut, incoming.ClockOut)
	if err != nil {
		return err
	}
	if clockInActions+clockOutActions != 1 {
		return repositories.ErrConflict
	}
	return nil
}

func validateReviewAppend(current, incoming *attendancePunchDocument) (int, error) {
	if current == nil || incoming == nil {
		if current == nil && incoming == nil {
			return 0, nil
		}
		return 0, repositories.ErrConflict
	}
	if current.ID != incoming.ID || current.Kind != incoming.Kind || current.SubmissionID != incoming.SubmissionID || !current.RecordedAt.Equal(incoming.RecordedAt) || current.Location.EvidenceRef != incoming.Location.EvidenceRef {
		return 0, repositories.ErrConflict
	}
	if len(incoming.History) < len(current.History) || len(incoming.History) > len(current.History)+1 || !sameStoredReviewPrefix(current.History, incoming.History) {
		return 0, repositories.ErrConflict
	}
	if len(incoming.History) == len(current.History) {
		if !sameStoredPunch(current, incoming) {
			return 0, repositories.ErrConflict
		}
		return 0, nil
	}
	appended := incoming.History[len(current.History)]
	before := attendanceReviewSnapshotDocument{EffectiveAt: current.EffectiveAt, Location: current.Location, TimeAnomaly: current.TimeAnomaly}
	after := attendanceReviewSnapshotDocument{EffectiveAt: incoming.EffectiveAt, Location: incoming.Location, TimeAnomaly: incoming.TimeAnomaly}
	if !sameStoredReviewSnapshot(appended.Before, before) || !sameStoredReviewSnapshot(appended.After, after) || appended.State != incoming.ReviewState {
		return 0, repositories.ErrConflict
	}
	return 1, nil
}

func sameStoredPunch(left, right *attendancePunchDocument) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if left.ID != right.ID || left.Kind != right.Kind || left.SubmissionID != right.SubmissionID ||
		!left.RecordedAt.Equal(right.RecordedAt) || !left.EffectiveAt.Equal(right.EffectiveAt) ||
		left.Location != right.Location || left.TimeAnomaly != right.TimeAnomaly ||
		left.ReviewState != right.ReviewState || len(left.History) != len(right.History) {
		return false
	}
	return sameStoredReviewPrefix(left.History, right.History)
}

func sameStoredReviewPrefix(prefix, values []attendanceReviewDocument) bool {
	if len(prefix) > len(values) {
		return false
	}
	for i := range prefix {
		left, right := prefix[i], values[i]
		if left.State != right.State || left.Provenance != right.Provenance || !left.At.Equal(right.At) ||
			left.Reviewer != right.Reviewer || left.Reason != right.Reason ||
			!sameStoredReviewSnapshot(left.Before, right.Before) || !sameStoredReviewSnapshot(left.After, right.After) {
			return false
		}
	}
	return true
}

func sameStoredReviewSnapshot(left, right attendanceReviewSnapshotDocument) bool {
	return left.EffectiveAt.Equal(right.EffectiveAt) && left.Location == right.Location && left.TimeAnomaly == right.TimeAnomaly
}

func sameStoredParticipant(left, right attendanceParticipantDocument) bool {
	return left.SchemaVersion == right.SchemaVersion && left.EmployeeID == right.EmployeeID &&
		left.EmployeeCode == right.EmployeeCode && left.DisplayName == right.DisplayName &&
		left.Active == right.Active && left.ParticipantID == right.ParticipantID &&
		sameOptionalTime(left.DeactivatedAt, right.DeactivatedAt) && sameOptionalTime(left.ExpiresAt, right.ExpiresAt)
}

func sameStoredWorkplace(left, right attendanceWorkplaceDocument) bool {
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return left == right
}

func sameStoredShift(left, right attendanceShiftDocument) bool {
	return left.SchemaVersion == right.SchemaVersion && left.ID == right.ID && left.WorkplaceID == right.WorkplaceID &&
		left.Label == right.Label && left.Active == right.Active && left.StartsAt.Equal(right.StartsAt) &&
		left.EndsAt.Equal(right.EndsAt) && left.ClockInOpensAt.Equal(right.ClockInOpensAt) &&
		left.ClockInClosesAt.Equal(right.ClockInClosesAt) && left.ClockOutOpensAt.Equal(right.ClockOutOpensAt) &&
		left.ClockOutClosesAt.Equal(right.ClockOutClosesAt) && left.SubmissionOpensAt.Equal(right.SubmissionOpensAt) &&
		left.SubmissionClosesAt.Equal(right.SubmissionClosesAt) && left.PolicyVersion == right.PolicyVersion
}

func sameStoredAssignment(left, right attendanceAssignmentDocument) bool {
	return left.SchemaVersion == right.SchemaVersion && left.ID == right.ID && left.EmployeeID == right.EmployeeID &&
		left.ShiftInstanceID == right.ShiftInstanceID && left.Active == right.Active &&
		left.SubmissionOpensAt.Equal(right.SubmissionOpensAt) && left.SubmissionClosesAt.Equal(right.SubmissionClosesAt)
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func (s *AttendanceStore) ListReviewRequired(ctx context.Context, limit int) ([]aggregates.Attendance, error) {
	if ctx == nil || limit <= 0 || limit > 500 {
		return nil, errors.New("attendance review limit must be 1..500")
	}
	base := s.client.Collection(attendanceRecordsCollection).Where("review_required", "==", true).OrderBy("updated_at", sdk.Asc)
	batchSize := limit
	query := base.Limit(batchSize)
	result := make([]aggregates.Attendance, 0, limit)
	now := s.now().UTC()
	for len(result) < limit {
		iter := query.Documents(ctx)
		var last *sdk.DocumentSnapshot
		pageCount := 0
		for len(result) < limit {
			snap, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, attendanceStoreError(ctx, "list reviews")
			}
			last, pageCount = snap, pageCount+1
			var stored attendanceRecordDocument
			if err := snap.DataTo(&stored); err != nil {
				iter.Stop()
				return nil, ErrRepositoryCorrupt
			}
			if stored.ExpiresAt.IsZero() {
				iter.Stop()
				return nil, ErrRepositoryCorrupt
			}
			if retentionExpired(stored.ExpiresAt, now) {
				continue
			}
			value, err := restoreAttendanceRecord(stored)
			if err != nil {
				iter.Stop()
				return nil, err
			}
			result = append(result, value)
		}
		iter.Stop()
		if len(result) >= limit || pageCount < batchSize || last == nil {
			break
		}
		query = base.StartAfter(last).Limit(batchSize)
	}
	return result, nil
}

func punchRequiresReview(p *attendancePunchDocument) bool {
	return p != nil && p.ReviewState == string(valueobjects.ReviewRequired)
}
func retentionBase(d attendanceRecordDocument) time.Time {
	if d.ClockIn != nil {
		return d.ClockIn.RecordedAt
	}
	if d.ClockOut != nil {
		return d.ClockOut.RecordedAt
	}
	return d.UpdatedAt
}

func newPunchForRevision(v aggregates.Attendance, expected uint64) (entities.Punch, bool) {
	if v.Revision() != expected+1 {
		return entities.Punch{}, false
	}
	if p, ok := v.ClockOut(); ok && expected > 0 {
		return p, true
	}
	if p, ok := v.ClockIn(); ok && expected == 0 {
		return p, true
	}
	return entities.Punch{}, false
}
func reviewActionCounts(d attendanceRecordDocument) map[string]int {
	m := map[string]int{}
	if d.ClockIn != nil {
		m[d.ClockIn.ID] = len(d.ClockIn.History)
	}
	if d.ClockOut != nil {
		m[d.ClockOut.ID] = len(d.ClockOut.History)
	}
	return m
}
func (s *AttendanceStore) createReviewActions(tx *sdk.Transaction, attendanceID string, d attendanceRecordDocument, current map[string]int) error {
	for _, p := range []*attendancePunchDocument{d.ClockIn, d.ClockOut} {
		if p == nil {
			continue
		}
		start := current[p.ID]
		if start > len(p.History) {
			return ErrRepositoryCorrupt
		}
		for i := start; i < len(p.History); i++ {
			a := p.History[i]
			ref := s.client.Collection(attendanceReviewsCollection).Doc(attendanceDocumentID("review", attendanceID, p.ID, fmt.Sprint(i)))
			doc := attendanceReviewActionDocument{
				SchemaVersion: attendanceSchemaVersion, AttendanceID: attendanceID, PunchID: p.ID, Sequence: i,
				State: a.State, Provenance: a.Provenance, At: a.At, Reviewer: a.Reviewer, Reason: a.Reason,
				Before: a.Before, After: a.After, ExpiresAt: d.ExpiresAt,
			}
			if err := tx.Create(ref, doc); err != nil {
				return err
			}
		}
	}
	return nil
}
