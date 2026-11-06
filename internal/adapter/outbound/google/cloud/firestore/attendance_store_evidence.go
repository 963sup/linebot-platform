package firestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AttendanceStore) findEvidence(ctx context.Context, evidenceRef valueobjects.EvidenceRef) (valueobjects.LocationEvidence, error) {
	if ctx == nil || !evidenceRef.Valid() {
		return valueobjects.LocationEvidence{}, errors.New("valid evidence reference is required")
	}
	punchID, err := valueobjects.NewPunchID(string(evidenceRef))
	if err != nil {
		return valueobjects.LocationEvidence{}, errors.New("valid evidence reference is required")
	}
	snap, err := s.client.Collection(attendanceEvidenceCollection).Doc(attendanceDocumentID("location-evidence", string(punchID))).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return valueobjects.LocationEvidence{}, repositories.ErrNotFound
	}
	if err != nil {
		return valueobjects.LocationEvidence{}, attendanceStoreError(ctx, "read location evidence")
	}
	var stored attendanceEvidenceDocument
	if err := snap.DataTo(&stored); err != nil || stored.SchemaVersion != attendanceSchemaVersion || stored.PunchID != string(punchID) || stored.ExpiresAt.IsZero() {
		return valueobjects.LocationEvidence{}, ErrRepositoryCorrupt
	}
	if retentionExpired(stored.ExpiresAt, s.now().UTC()) {
		return valueobjects.LocationEvidence{}, repositories.ErrNotFound
	}
	plaintext, err := s.keyring.Open(string(punchID), stored.KeyID, stored.Ciphertext)
	if err != nil {
		return valueobjects.LocationEvidence{}, fmt.Errorf("%w: decrypt location evidence", ErrRepositoryCorrupt)
	}
	var decoded struct{ Latitude, Longitude, AccuracyMeters float64 }
	if err := json.Unmarshal(plaintext, &decoded); err != nil {
		return valueobjects.LocationEvidence{}, ErrRepositoryCorrupt
	}
	return valueobjects.NewLocationEvidence(decoded.Latitude, decoded.Longitude, decoded.AccuracyMeters)
}

// findEvidenceForReview records an immutable access audit before decrypting
// short-lived raw coordinates. Only the audited Application port exposes it.
func (s *AttendanceStore) findEvidenceForReview(ctx context.Context, evidenceRef valueobjects.EvidenceRef, reviewer, reason string, accessedAt time.Time) (valueobjects.LocationEvidence, error) {
	if ctx == nil || !evidenceRef.Valid() || strings.TrimSpace(reviewer) == "" || strings.TrimSpace(reason) == "" || accessedAt.IsZero() {
		return valueobjects.LocationEvidence{}, errors.New("attendance evidence review identity and reason are required")
	}
	accessedAt = accessedAt.UTC()
	doc := attendanceEvidenceAccessDocument{SchemaVersion: attendanceSchemaVersion, EvidenceRef: string(evidenceRef), Reviewer: reviewer, Reason: reason, AccessedAt: accessedAt, ExpiresAt: attendanceExpiry(accessedAt)}
	if _, err := s.client.Collection(attendanceEvidenceAccessCollection).NewDoc().Create(ctx, doc); err != nil {
		return valueobjects.LocationEvidence{}, classifyAttendanceStoreError(ctx, err)
	}
	return s.findEvidence(ctx, evidenceRef)
}

// ReadEvidence implements the Application review port. Cloud Audit Logs retain
// the authenticated ADC caller; this record additionally captures the explicit
// business access reason and operator-supplied reviewer claim.
func (s *AttendanceStore) ReadEvidence(ctx context.Context, evidenceRef valueobjects.EvidenceRef, reviewer, accessReason string, accessedAt time.Time) (valueobjects.LocationEvidence, error) {
	return s.findEvidenceForReview(ctx, evidenceRef, reviewer, accessReason, accessedAt)
}
