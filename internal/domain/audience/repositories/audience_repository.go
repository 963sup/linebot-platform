// Package repositories defines technology-free Audience persistence contracts.
package repositories

import (
	"context"

	"github.com/963sup/linebot-platform/internal/domain/audience/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
)

type AudienceRepository interface {
	Find(context.Context, valueobjects.AudienceID) (aggregates.AudienceDefinition, error)
	Save(context.Context, aggregates.AudienceDefinition) error
}
