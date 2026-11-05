// Package promotion orchestrates the Promotion bounded context.
package promotion

import (
	"context"
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/promotion/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/promotion/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
)

var ErrContextRequired = errors.New("context is required")

type Service struct {
	repository repositories.PromotionRepository
}

func NewService(repository repositories.PromotionRepository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("promotion repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) Define(ctx context.Context, id valueobjects.PromotionID, offer valueobjects.OfferReference, policy valueobjects.EligibilityPolicyReference, period valueobjects.Period) (aggregates.Promotion, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.Promotion{}, err
	}
	promotion, err := aggregates.NewPromotion(id, offer, policy, period)
	if err != nil {
		return aggregates.Promotion{}, err
	}
	if err := s.repository.Save(ctx, promotion); err != nil {
		return aggregates.Promotion{}, err
	}
	promotion.AcknowledgeEvents()
	return promotion, nil
}

func (s *Service) Activate(ctx context.Context, id valueobjects.PromotionID, at time.Time) (aggregates.Promotion, error) {
	return s.change(ctx, id, func(promotion *aggregates.Promotion) error {
		return promotion.Activate(at)
	})
}

func (s *Service) Close(ctx context.Context, id valueobjects.PromotionID) (aggregates.Promotion, error) {
	return s.change(ctx, id, func(promotion *aggregates.Promotion) error {
		return promotion.Close()
	})
}

func (s *Service) CreateDeliveryIntent(ctx context.Context, id valueobjects.PromotionID, recipient string, decision valueobjects.EligibilityDecision, at time.Time) (valueobjects.DeliveryIntent, error) {
	if err := validateContext(ctx); err != nil {
		return valueobjects.DeliveryIntent{}, err
	}
	promotion, err := s.repository.Find(ctx, id)
	if err != nil {
		return valueobjects.DeliveryIntent{}, err
	}
	intent, err := promotion.CreateDeliveryIntent(recipient, decision, at)
	if err != nil {
		return valueobjects.DeliveryIntent{}, err
	}
	// DeliveryIntentCreated is modeled for the application boundary, but this
	// context does not publish an integration event in the current baseline.
	// Persist the aggregate before acknowledging the modeled event.
	if err := s.repository.Save(ctx, promotion); err != nil {
		return valueobjects.DeliveryIntent{}, err
	}
	promotion.AcknowledgeEvents()
	return intent, nil
}

func (s *Service) change(ctx context.Context, id valueobjects.PromotionID, change func(*aggregates.Promotion) error) (aggregates.Promotion, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.Promotion{}, err
	}
	promotion, err := s.repository.Find(ctx, id)
	if err != nil {
		return aggregates.Promotion{}, err
	}
	if err := change(&promotion); err != nil {
		return aggregates.Promotion{}, err
	}
	if err := s.repository.Save(ctx, promotion); err != nil {
		return aggregates.Promotion{}, err
	}
	promotion.AcknowledgeEvents()
	return promotion, nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}
