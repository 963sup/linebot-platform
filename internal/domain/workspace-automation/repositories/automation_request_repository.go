// Package repositories contains domain-owned, technology-free contracts.
package repositories

import (
	"context"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

// AutomationRequestRepository persists the AutomationRequest aggregate.
type AutomationRequestRepository interface {
	Find(context.Context, valueobjects.AutomationRequestID) (aggregates.AutomationRequest, error)
	Save(context.Context, aggregates.AutomationRequest) error
}
