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
	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const interactionSurfacesCollection = "interaction_surfaces_v1"

type storedRegion struct {
	ID              string `firestore:"id"`
	ActionReference string `firestore:"action_reference"`
	AssetReference  string `firestore:"asset_reference"`
}

type storedInteractionSurface struct {
	SchemaVersion int            `firestore:"schema_version"`
	ID            string         `firestore:"id"`
	Regions       []storedRegion `firestore:"regions"`
	Version       int64          `firestore:"version"`
	Status        uint8          `firestore:"status"`
	CreatedAt     time.Time      `firestore:"created_at"`
	UpdatedAt     time.Time      `firestore:"updated_at"`
}

// InteractionSurfaceRepository persists desired presentation aggregates.
type InteractionSurfaceRepository struct {
	client *sdk.Client
	now    func() time.Time
}

func NewInteractionSurfaceRepository(client *sdk.Client) (*InteractionSurfaceRepository, error) {
	if client == nil {
		return nil, errors.New("Firestore client is required")
	}
	return &InteractionSurfaceRepository{client: client, now: time.Now}, nil
}

func (r *InteractionSurfaceRepository) Find(ctx context.Context, id valueobjects.SurfaceID) (aggregates.InteractionSurface, error) {
	if ctx == nil || !id.Valid() {
		return aggregates.InteractionSurface{}, errors.New("valid interaction surface repository input is required")
	}
	if err := ctx.Err(); err != nil {
		return aggregates.InteractionSurface{}, err
	}
	snapshot, err := r.document(id.String()).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return aggregates.InteractionSurface{}, ErrRepositoryNotFound
	}
	if err != nil {
		return aggregates.InteractionSurface{}, surfaceInfrastructureError(ctx, "read")
	}
	var stored storedInteractionSurface
	if err := snapshot.DataTo(&stored); err != nil {
		return aggregates.InteractionSurface{}, fmt.Errorf("%w: decoding interaction surface", ErrRepositoryCorrupt)
	}
	value, err := restoreInteractionSurface(stored)
	if err != nil {
		return aggregates.InteractionSurface{}, fmt.Errorf("%w: invalid interaction surface document", ErrRepositoryCorrupt)
	}
	if value.ID() != id {
		return aggregates.InteractionSurface{}, fmt.Errorf("%w: surface id mismatch", ErrRepositoryCorrupt)
	}
	return value, nil
}

func (r *InteractionSurfaceRepository) Save(ctx context.Context, value aggregates.InteractionSurface) error {
	if ctx == nil || !value.ID().Valid() || !value.Version().Valid() || uint64(value.Version()) > math.MaxInt64 || !value.Status().Valid() {
		return errors.New("valid interaction surface is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ref := r.document(value.ID().String())
	now := r.now().UTC()
	incoming := storeInteractionSurface(value, now)
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
		var current storedInteractionSurface
		if err := snapshot.DataTo(&current); err != nil {
			return fmt.Errorf("%w: decoding interaction surface", ErrRepositoryCorrupt)
		}
		if err := validateSurfaceRevision(current, incoming); err != nil {
			return err
		}
		document := incoming
		document.CreatedAt = current.CreatedAt
		if document.UpdatedAt.Before(document.CreatedAt) {
			document.UpdatedAt = document.CreatedAt
		}
		return tx.Set(ref, document)
	})
	return classifySurfaceSaveError(ctx, err)
}

func (r *InteractionSurfaceRepository) document(id string) *sdk.DocumentRef {
	return r.client.Collection(interactionSurfacesCollection).Doc(interactionSurfaceDocumentID(id))
}

func interactionSurfaceDocumentID(id string) string {
	digest := sha256.Sum256([]byte("interaction-surface-v1\x00" + id))
	return hex.EncodeToString(digest[:])
}

func storeInteractionSurface(value aggregates.InteractionSurface, now time.Time) storedInteractionSurface {
	regions := make([]storedRegion, 0, len(value.Presentation().Regions()))
	for _, region := range value.Presentation().Regions() {
		regions = append(regions, storedRegion{ID: region.ID(), ActionReference: region.ActionReference(), AssetReference: region.AssetReference()})
	}
	return storedInteractionSurface{
		SchemaVersion: 1, ID: value.ID().String(), Regions: regions,
		Version: int64(value.Version()), Status: uint8(value.Status()), UpdatedAt: now,
	}
}

func restoreInteractionSurface(stored storedInteractionSurface) (aggregates.InteractionSurface, error) {
	if err := validateStoredSurface(stored); err != nil {
		return aggregates.InteractionSurface{}, err
	}
	id, err := valueobjects.NewSurfaceID(stored.ID)
	if err != nil {
		return aggregates.InteractionSurface{}, err
	}
	regions := make([]valueobjects.Region, 0, len(stored.Regions))
	for _, saved := range stored.Regions {
		region, err := valueobjects.NewRegion(saved.ID, saved.ActionReference, saved.AssetReference)
		if err != nil {
			return aggregates.InteractionSurface{}, err
		}
		regions = append(regions, region)
	}
	presentation, err := valueobjects.NewPresentation(regions)
	if err != nil {
		return aggregates.InteractionSurface{}, err
	}
	return aggregates.RestoreInteractionSurface(id, presentation, valueobjects.Version(uint64(stored.Version)), aggregates.Status(stored.Status))
}

func validateStoredSurface(stored storedInteractionSurface) error {
	if stored.SchemaVersion != 1 || stored.Version <= 0 || len(stored.Regions) == 0 || stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() || stored.UpdatedAt.Before(stored.CreatedAt) {
		return ErrRepositoryCorrupt
	}
	return nil
}

func validateSurfaceRevision(current, incoming storedInteractionSurface) error {
	if _, err := restoreInteractionSurface(current); err != nil {
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

func classifySurfaceSaveError(ctx context.Context, err error) error {
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
		return surfaceInfrastructureError(ctx, "save")
	}
}

func surfaceInfrastructureError(ctx context.Context, operation string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: interaction surface %s failed", ErrRepositoryUnavailable, operation)
}

var _ repositories.InteractionSurfaceRepository = (*InteractionSurfaceRepository)(nil)
