// Package repositories defines technology-free presentation persistence contracts.
package repositories

import (
	"context"

	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
)

type InteractionSurfaceRepository interface {
	Find(context.Context, valueobjects.SurfaceID) (aggregates.InteractionSurface, error)
	Save(context.Context, aggregates.InteractionSurface) error
}
