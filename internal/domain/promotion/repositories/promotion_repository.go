// Package repositories defines technology-free Promotion persistence contracts.
package repositories

import (
	"context"

	"github.com/963sup/linebot-platform/internal/domain/promotion/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
)

type PromotionRepository interface {
	Find(context.Context, valueobjects.PromotionID) (aggregates.Promotion, error)
	// Save atomically persists aggregate state.
	Save(context.Context, aggregates.Promotion) error
}
