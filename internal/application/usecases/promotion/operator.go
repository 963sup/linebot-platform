package promotion

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/promotion/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/promotion/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
)

type OperatorSummary struct {
	ID                      string
	OfferSHA256             string
	EligibilityPolicySHA256 string
	StartsAt                time.Time
	EndsAt                  time.Time
	Revision                uint64
	Status                  string
}

type DeliveryIntentSummary struct {
	PromotionID     string
	RecipientSHA256 string
	OfferSHA256     string
}

type Operator struct {
	repository repositories.PromotionRepository
	service    *Service
}

func NewOperator(repository repositories.PromotionRepository) (*Operator, error) {
	if repository == nil {
		return nil, errors.New("promotion repository is required")
	}
	service, err := NewService(repository)
	if err != nil {
		return nil, err
	}
	return &Operator{repository: repository, service: service}, nil
}

func (o *Operator) Get(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewPromotionID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.repository.Find(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizePromotion(value), nil
}

func (o *Operator) Define(ctx context.Context, rawID, offer, policy string, startsAt, endsAt time.Time) (OperatorSummary, error) {
	id, err := valueobjects.NewPromotionID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	offerReference, err := valueobjects.NewOfferReference(offer)
	if err != nil {
		return OperatorSummary{}, err
	}
	policyReference, err := valueobjects.NewEligibilityPolicyReference(policy)
	if err != nil {
		return OperatorSummary{}, err
	}
	period, err := valueobjects.NewPeriod(startsAt, endsAt)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Define(ctx, id, offerReference, policyReference, period)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizePromotion(value), nil
}

func (o *Operator) Activate(ctx context.Context, rawID string, at time.Time) (OperatorSummary, error) {
	id, err := valueobjects.NewPromotionID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Activate(ctx, id, at)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizePromotion(value), nil
}

func (o *Operator) Close(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewPromotionID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Close(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizePromotion(value), nil
}

func (o *Operator) CreateDeliveryIntent(ctx context.Context, rawID, recipient, policy string, eligible bool, at time.Time) (DeliveryIntentSummary, error) {
	id, err := valueobjects.NewPromotionID(rawID)
	if err != nil {
		return DeliveryIntentSummary{}, err
	}
	policyReference, err := valueobjects.NewEligibilityPolicyReference(policy)
	if err != nil {
		return DeliveryIntentSummary{}, err
	}
	decision, err := valueobjects.NewEligibilityDecision(policyReference, eligible)
	if err != nil {
		return DeliveryIntentSummary{}, err
	}
	intent, err := o.service.CreateDeliveryIntent(ctx, id, recipient, decision, at)
	if err != nil {
		return DeliveryIntentSummary{}, err
	}
	return DeliveryIntentSummary{
		PromotionID:     rawID,
		RecipientSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(intent.RecipientReference()))),
		OfferSHA256:     fmt.Sprintf("%x", sha256.Sum256([]byte(intent.OfferReference().String()))),
	}, nil
}

func summarizePromotion(value aggregates.Promotion) OperatorSummary {
	return OperatorSummary{
		ID:                      value.ID().String(),
		OfferSHA256:             fmt.Sprintf("%x", sha256.Sum256([]byte(value.Offer().String()))),
		EligibilityPolicySHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(value.EligibilityPolicy().String()))),
		StartsAt:                value.Period().Start(), EndsAt: value.Period().End(),
		Revision: value.Revision(), Status: promotionStatus(value.Status()),
	}
}

func promotionStatus(status aggregates.Status) string {
	switch status {
	case aggregates.StatusDraft:
		return "draft"
	case aggregates.StatusActive:
		return "active"
	case aggregates.StatusClosed:
		return "closed"
	default:
		return "unknown"
	}
}
