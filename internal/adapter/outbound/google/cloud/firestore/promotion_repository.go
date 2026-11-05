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
	"github.com/963sup/linebot-platform/internal/domain/promotion/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/promotion/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const promotionsCollection = "promotions_v1"

type storedPromotion struct {
	SchemaVersion              int       `firestore:"schema_version"`
	ID                         string    `firestore:"id"`
	OfferReference             string    `firestore:"offer_reference"`
	EligibilityPolicyReference string    `firestore:"eligibility_policy_reference"`
	StartsAt                   time.Time `firestore:"starts_at"`
	EndsAt                     time.Time `firestore:"ends_at"`
	Status                     uint8     `firestore:"status"`
	Revision                   int64     `firestore:"revision"`
	CreatedAt                  time.Time `firestore:"created_at"`
	UpdatedAt                  time.Time `firestore:"updated_at"`
}

// PromotionRepository persists provider-independent Promotion aggregates.
// Its Firestore client is caller-owned and must outlive this repository.
type PromotionRepository struct {
	client *sdk.Client
	now    func() time.Time
}

func NewPromotionRepository(client *sdk.Client) (*PromotionRepository, error) {
	if client == nil {
		return nil, errors.New("Firestore client is required")
	}
	return &PromotionRepository{client: client, now: time.Now}, nil
}

func (r *PromotionRepository) Find(ctx context.Context, id valueobjects.PromotionID) (aggregates.Promotion, error) {
	if ctx == nil || !id.Valid() {
		return aggregates.Promotion{}, errors.New("valid promotion repository input is required")
	}
	if err := ctx.Err(); err != nil {
		return aggregates.Promotion{}, err
	}

	snapshot, err := r.document(id.String()).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return aggregates.Promotion{}, ErrRepositoryNotFound
	}
	if err != nil {
		return aggregates.Promotion{}, promotionInfrastructureError(ctx, "read")
	}
	var stored storedPromotion
	if err := snapshot.DataTo(&stored); err != nil {
		return aggregates.Promotion{}, fmt.Errorf("%w: decoding promotion", ErrRepositoryCorrupt)
	}
	value, err := restorePromotion(stored)
	if err != nil {
		return aggregates.Promotion{}, fmt.Errorf("%w: invalid promotion document", ErrRepositoryCorrupt)
	}
	if value.ID() != id {
		return aggregates.Promotion{}, fmt.Errorf("%w: promotion id mismatch", ErrRepositoryCorrupt)
	}
	return value, nil
}

func (r *PromotionRepository) Save(ctx context.Context, value aggregates.Promotion) error {
	if ctx == nil || !value.ID().Valid() || !value.Status().Valid() || value.Revision() == 0 || value.Revision() > math.MaxInt64 {
		return errors.New("valid promotion is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	ref := r.document(value.ID().String())
	now := r.now().UTC()
	incoming := storePromotion(value, now)
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
		var current storedPromotion
		if err := snapshot.DataTo(&current); err != nil {
			return fmt.Errorf("%w: decoding promotion", ErrRepositoryCorrupt)
		}
		if err := validatePromotionRevision(current, incoming); err != nil {
			return err
		}
		incoming.CreatedAt = current.CreatedAt
		return tx.Set(ref, incoming)
	})
	return classifyPromotionSaveError(ctx, err)
}

func (r *PromotionRepository) document(id string) *sdk.DocumentRef {
	return r.client.Collection(promotionsCollection).Doc(promotionDocumentID(id))
}

func promotionDocumentID(id string) string {
	digest := sha256.Sum256([]byte("promotion-v1\x00" + id))
	return hex.EncodeToString(digest[:])
}

func storePromotion(value aggregates.Promotion, now time.Time) storedPromotion {
	return storedPromotion{
		SchemaVersion:              1,
		ID:                         value.ID().String(),
		OfferReference:             value.Offer().String(),
		EligibilityPolicyReference: value.EligibilityPolicy().String(),
		StartsAt:                   value.Period().Start(),
		EndsAt:                     value.Period().End(),
		Status:                     uint8(value.Status()),
		Revision:                   int64(value.Revision()),
		UpdatedAt:                  now,
	}
}

func restorePromotion(stored storedPromotion) (aggregates.Promotion, error) {
	if err := validateStoredPromotion(stored); err != nil {
		return aggregates.Promotion{}, err
	}
	id, err := valueobjects.NewPromotionID(stored.ID)
	if err != nil {
		return aggregates.Promotion{}, err
	}
	offer, err := valueobjects.NewOfferReference(stored.OfferReference)
	if err != nil {
		return aggregates.Promotion{}, err
	}
	policy, err := valueobjects.NewEligibilityPolicyReference(stored.EligibilityPolicyReference)
	if err != nil {
		return aggregates.Promotion{}, err
	}
	period, err := valueobjects.NewPeriod(stored.StartsAt, stored.EndsAt)
	if err != nil {
		return aggregates.Promotion{}, err
	}
	return aggregates.RestorePromotion(id, offer, policy, period, aggregates.Status(stored.Status), uint64(stored.Revision))
}

func validateStoredPromotion(stored storedPromotion) error {
	if stored.SchemaVersion != 1 || stored.Revision <= 0 || stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() || stored.UpdatedAt.Before(stored.CreatedAt) {
		return ErrRepositoryCorrupt
	}
	return nil
}

func validatePromotionRevision(current, incoming storedPromotion) error {
	if err := validateStoredPromotion(current); err != nil {
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
	if current.OfferReference != incoming.OfferReference ||
		current.EligibilityPolicyReference != incoming.EligibilityPolicyReference ||
		!current.StartsAt.Equal(incoming.StartsAt) || !current.EndsAt.Equal(incoming.EndsAt) {
		return ErrRepositoryConflict
	}
	return nil
}

func classifyPromotionSaveError(ctx context.Context, err error) error {
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
		return promotionInfrastructureError(ctx, "save")
	}
}

func promotionInfrastructureError(ctx context.Context, operation string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: promotion %s failed", ErrRepositoryUnavailable, operation)
}

var _ repositories.PromotionRepository = (*PromotionRepository)(nil)
