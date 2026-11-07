package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	attendanceParticipantsCollection     = "attendance_participants_v1"
	attendanceParticipantBindings        = "attendance_participant_bindings_v1"
	attendanceEnrollmentTokensCollection = "attendance_enrollment_tokens_v1"
	attendanceDaysCollection             = "attendance_days_v2"
	attendanceParticipantSchemaVersion   = 1
	attendanceDaySchemaVersion           = 2
)

// AttendanceStore owns only participant enrollment and one AttendanceDay CAS.
// The injected Firestore client remains owned by the composition root.
type AttendanceStore struct {
	client *sdk.Client
	now    func() time.Time
}

func NewAttendanceStore(client *sdk.Client) (*AttendanceStore, error) {
	if client == nil {
		return nil, errors.New("attendance store requires Firestore")
	}
	return &AttendanceStore{client: client, now: time.Now}, nil
}

func attendanceDocumentID(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(h, "%d:", len(part))
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func retentionExpired(expiresAt, now time.Time) bool {
	return !expiresAt.IsZero() && !expiresAt.After(now)
}

func classifyAttendanceStoreError(ctx context.Context, err error) error {
	if err == nil || errors.Is(err, repositories.ErrNotFound) || errors.Is(err, repositories.ErrConflict) || errors.Is(err, repositories.ErrConcurrentUpdate) || errors.Is(err, ErrRepositoryCorrupt) {
		return err
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	switch status.Code(err) {
	case codes.NotFound:
		return repositories.ErrNotFound
	case codes.AlreadyExists:
		return repositories.ErrConflict
	case codes.Aborted, codes.FailedPrecondition:
		return repositories.ErrConcurrentUpdate
	default:
		return attendanceStoreError(ctx, "operation")
	}
}

func attendanceStoreError(ctx context.Context, operation string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: attendance %s failed", ErrRepositoryUnavailable, operation)
}
