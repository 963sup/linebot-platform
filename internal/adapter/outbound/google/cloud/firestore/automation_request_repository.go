package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const automationRequestsCollection = "automation_requests_v1"

type storedAutomationRequest struct {
	SchemaVersion  int       `firestore:"schema_version"`
	ID             string    `firestore:"id"`
	Action         string    `firestore:"action"`
	RequiredScopes []string  `firestore:"required_scopes"`
	ApprovedBy     string    `firestore:"approved_by,omitempty"`
	Output         string    `firestore:"output,omitempty"`
	Status         string    `firestore:"status"`
	Revision       int64     `firestore:"revision"`
	CreatedAt      time.Time `firestore:"created_at"`
	UpdatedAt      time.Time `firestore:"updated_at"`
}

// AutomationRequestRepository persists provider-neutral Workspace Automation
// aggregates. Its Firestore client is caller-owned.
type AutomationRequestRepository struct {
	client *sdk.Client
	now    func() time.Time
}

func NewAutomationRequestRepository(client *sdk.Client) (*AutomationRequestRepository, error) {
	if client == nil {
		return nil, errors.New("Firestore client is required")
	}
	return &AutomationRequestRepository{client: client, now: time.Now}, nil
}

func (r *AutomationRequestRepository) Find(ctx context.Context, id valueobjects.AutomationRequestID) (aggregates.AutomationRequest, error) {
	if ctx == nil || !id.Valid() {
		return aggregates.AutomationRequest{}, errors.New("valid automation request repository input is required")
	}
	if err := ctx.Err(); err != nil {
		return aggregates.AutomationRequest{}, err
	}

	snapshot, err := r.document(id.String()).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return aggregates.AutomationRequest{}, ErrRepositoryNotFound
	}
	if err != nil {
		return aggregates.AutomationRequest{}, automationInfrastructureError(ctx, "read")
	}
	var stored storedAutomationRequest
	if err := snapshot.DataTo(&stored); err != nil {
		return aggregates.AutomationRequest{}, fmt.Errorf("%w: decoding automation request", ErrRepositoryCorrupt)
	}
	value, err := restoreAutomationRequest(stored)
	if err != nil {
		return aggregates.AutomationRequest{}, fmt.Errorf("%w: invalid automation request document", ErrRepositoryCorrupt)
	}
	if value.ID() != id {
		return aggregates.AutomationRequest{}, fmt.Errorf("%w: automation request id mismatch", ErrRepositoryCorrupt)
	}
	return value, nil
}

func (r *AutomationRequestRepository) Save(ctx context.Context, value aggregates.AutomationRequest) error {
	if ctx == nil || !value.ID().Valid() || !value.Status().Valid() || value.Revision() == 0 || value.Revision() > math.MaxInt64 {
		return errors.New("valid automation request is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	ref := r.document(value.ID().String())
	now := r.now().UTC()
	incoming := storeAutomationRequest(value, now)
	err := r.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			if incoming.Revision != 1 {
				return ErrRepositoryConflict
			}
			incoming.CreatedAt = now
			return tx.Create(ref, incoming)
		}
		if err != nil {
			return err
		}
		var current storedAutomationRequest
		if err := snapshot.DataTo(&current); err != nil {
			return fmt.Errorf("%w: decoding automation request", ErrRepositoryCorrupt)
		}
		if err := validateAutomationRevision(current, incoming); err != nil {
			return err
		}
		incoming.CreatedAt = current.CreatedAt
		return tx.Set(ref, incoming)
	})
	return classifyAutomationSaveError(ctx, err)
}

func (r *AutomationRequestRepository) document(id string) *sdk.DocumentRef {
	return r.client.Collection(automationRequestsCollection).Doc(automationRequestDocumentID(id))
}

func automationRequestDocumentID(id string) string {
	digest := sha256.Sum256([]byte("automation-request-v1\x00" + id))
	return hex.EncodeToString(digest[:])
}

func storeAutomationRequest(value aggregates.AutomationRequest, now time.Time) storedAutomationRequest {
	return storedAutomationRequest{
		SchemaVersion:  1,
		ID:             value.ID().String(),
		Action:         value.Action().Name(),
		RequiredScopes: value.RequiredScopes(),
		ApprovedBy:     value.Approval().ApprovedBy(),
		Output:         value.Output().Detail(),
		Status:         string(value.Status()),
		Revision:       int64(value.Revision()),
		UpdatedAt:      now,
	}
}

func restoreAutomationRequest(stored storedAutomationRequest) (aggregates.AutomationRequest, error) {
	if err := validateStoredAutomationRequest(stored); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	id, err := valueobjects.NewAutomationRequestID(stored.ID)
	if err != nil {
		return aggregates.AutomationRequest{}, err
	}
	action, err := valueobjects.NewAction(stored.Action)
	if err != nil {
		return aggregates.AutomationRequest{}, err
	}
	scopes, err := valueobjects.NewRequiredScopes(stored.RequiredScopes)
	if err != nil {
		return aggregates.AutomationRequest{}, err
	}
	var approval valueobjects.Approval
	if stored.ApprovedBy != "" {
		approval, err = valueobjects.NewApproval(stored.ApprovedBy)
		if err != nil {
			return aggregates.AutomationRequest{}, err
		}
	}
	var output valueobjects.Output
	if stored.Output != "" {
		output, err = valueobjects.NewOutput(stored.Output)
		if err != nil {
			return aggregates.AutomationRequest{}, err
		}
	}
	return aggregates.RestoreAutomationRequest(id, action, scopes, approval, output, valueobjects.Status(stored.Status), uint64(stored.Revision))
}

func validateStoredAutomationRequest(stored storedAutomationRequest) error {
	if stored.SchemaVersion != 1 || stored.Revision <= 0 || stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() || stored.UpdatedAt.Before(stored.CreatedAt) {
		return ErrRepositoryCorrupt
	}
	return nil
}

func validateAutomationRevision(current, incoming storedAutomationRequest) error {
	if err := validateStoredAutomationRequest(current); err != nil {
		return err
	}
	if current.ID != incoming.ID {
		return ErrRepositoryCorrupt
	}
	if incoming.Revision == 1 {
		return ErrRepositoryConflict
	}
	if current.Revision != incoming.Revision-1 {
		return ErrConcurrentUpdate
	}
	if current.Action != incoming.Action || !slices.Equal(current.RequiredScopes, incoming.RequiredScopes) {
		return ErrRepositoryConflict
	}
	return nil
}

func classifyAutomationSaveError(ctx context.Context, err error) error {
	if err == nil || errors.Is(err, ErrRepositoryConflict) || errors.Is(err, ErrConcurrentUpdate) || errors.Is(err, ErrRepositoryCorrupt) {
		return err
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	switch status.Code(err) {
	case codes.AlreadyExists:
		return ErrRepositoryConflict
	case codes.Aborted, codes.FailedPrecondition:
		return ErrConcurrentUpdate
	default:
		return automationInfrastructureError(ctx, "save")
	}
}

func automationInfrastructureError(ctx context.Context, operation string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: automation request %s failed", ErrRepositoryUnavailable, operation)
}

var _ repositories.AutomationRequestRepository = (*AutomationRequestRepository)(nil)
