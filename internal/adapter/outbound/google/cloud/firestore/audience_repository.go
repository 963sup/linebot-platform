package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/audience/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/audience/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const audienceDefinitionsCollection = "audience_definitions_v1"

type storedAudienceDefinition struct {
	SchemaVersion      int       `firestore:"schema_version"`
	ID                 string    `firestore:"id"`
	CriteriaSource     string    `firestore:"criteria_source"`
	CriteriaDefinition string    `firestore:"criteria_definition"`
	Version            int64     `firestore:"version"`
	Status             uint8     `firestore:"status"`
	CreatedAt          time.Time `firestore:"created_at"`
	UpdatedAt          time.Time `firestore:"updated_at"`
}

// AudienceRepository persists provider-independent Audience aggregates.
type AudienceRepository struct {
	client *sdk.Client
	now    func() time.Time
}

func NewAudienceRepository(client *sdk.Client) (*AudienceRepository, error) {
	if client == nil {
		return nil, errors.New("Firestore client is required")
	}
	return &AudienceRepository{client: client, now: time.Now}, nil
}

func (r *AudienceRepository) Find(ctx context.Context, id valueobjects.AudienceID) (aggregates.AudienceDefinition, error) {
	if ctx == nil || !id.Valid() {
		return aggregates.AudienceDefinition{}, errors.New("valid audience repository input is required")
	}
	if err := ctx.Err(); err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	snapshot, err := r.document(id.String()).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return aggregates.AudienceDefinition{}, ErrRepositoryNotFound
	}
	if err != nil {
		return aggregates.AudienceDefinition{}, audienceInfrastructureError(ctx, "read")
	}
	var stored storedAudienceDefinition
	if err := snapshot.DataTo(&stored); err != nil {
		return aggregates.AudienceDefinition{}, fmt.Errorf("%w: decoding audience definition", ErrRepositoryCorrupt)
	}
	value, err := restoreAudienceDefinition(stored)
	if err != nil {
		return aggregates.AudienceDefinition{}, fmt.Errorf("%w: invalid audience definition document", ErrRepositoryCorrupt)
	}
	if value.ID() != id {
		return aggregates.AudienceDefinition{}, fmt.Errorf("%w: audience id mismatch", ErrRepositoryCorrupt)
	}
	return value, nil
}

func (r *AudienceRepository) Save(ctx context.Context, value aggregates.AudienceDefinition) error {
	if ctx == nil || !value.ID().Valid() || !value.Version().Valid() || uint64(value.Version()) > math.MaxInt64 || !value.Status().Valid() {
		return errors.New("valid audience definition is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ref := r.document(value.ID().String())
	now := r.now().UTC()
	incoming := storeAudienceDefinition(value, now)
	err := r.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			if value.Version() != valueobjects.InitialVersion {
				return ErrRepositoryConflict
			}
			document := incoming
			document.CreatedAt = now
			return tx.Create(ref, document)
		}
		if err != nil {
			return err
		}
		var current storedAudienceDefinition
		if err := snapshot.DataTo(&current); err != nil {
			return fmt.Errorf("%w: decoding audience definition", ErrRepositoryCorrupt)
		}
		if err := validateAudienceRevision(current, incoming); err != nil {
			return err
		}
		document := incoming
		document.CreatedAt = current.CreatedAt
		if document.UpdatedAt.Before(document.CreatedAt) {
			document.UpdatedAt = document.CreatedAt
		}
		return tx.Set(ref, document)
	})
	return classifyAudienceSaveError(ctx, err)
}

func (r *AudienceRepository) document(id string) *sdk.DocumentRef {
	return r.client.Collection(audienceDefinitionsCollection).Doc(audienceDocumentID(id))
}

func audienceDocumentID(id string) string {
	digest := sha256.Sum256([]byte("audience-definition-v1\x00" + id))
	return hex.EncodeToString(digest[:])
}

func storeAudienceDefinition(value aggregates.AudienceDefinition, now time.Time) storedAudienceDefinition {
	return storedAudienceDefinition{
		SchemaVersion: 1, ID: value.ID().String(), CriteriaSource: value.Criteria().Source(),
		CriteriaDefinition: value.Criteria().Definition(), Version: int64(value.Version()),
		Status: uint8(value.Status()), UpdatedAt: now,
	}
}

func restoreAudienceDefinition(stored storedAudienceDefinition) (aggregates.AudienceDefinition, error) {
	if err := validateStoredAudience(stored); err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	id, err := valueobjects.NewAudienceID(stored.ID)
	if err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	criteria, err := valueobjects.NewSelectionCriteria(stored.CriteriaSource, stored.CriteriaDefinition)
	if err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	return aggregates.RestoreAudienceDefinition(id, criteria, valueobjects.Version(uint64(stored.Version)), aggregates.Status(stored.Status))
}

func validateStoredAudience(stored storedAudienceDefinition) error {
	if stored.SchemaVersion != 1 || stored.Version <= 0 || stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() || stored.UpdatedAt.Before(stored.CreatedAt) {
		return ErrRepositoryCorrupt
	}
	return nil
}

func validateAudienceRevision(current, incoming storedAudienceDefinition) error {
	if _, err := restoreAudienceDefinition(current); err != nil {
		return ErrRepositoryCorrupt
	}
	if current.ID != incoming.ID {
		return ErrRepositoryCorrupt
	}
	if incoming.Version == 1 {
		return ErrRepositoryConflict
	}
	if current.Version != incoming.Version-1 {
		return ErrConcurrentUpdate
	}
	return nil
}

func classifyAudienceSaveError(ctx context.Context, err error) error {
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
		return audienceInfrastructureError(ctx, "save")
	}
}

func audienceInfrastructureError(ctx context.Context, operation string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: audience definition %s failed", ErrRepositoryUnavailable, operation)
}

var _ repositories.AudienceRepository = (*AudienceRepository)(nil)
