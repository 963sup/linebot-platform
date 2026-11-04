// Package repositories contains domain-owned, technology-free contracts.
package repositories

import (
	"context"

	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// ConnectionReader loads a domain account link without exposing credentials.
// Active-only backends may return an error for disconnected links.
type ConnectionReader interface {
	FindConnection(context.Context, valueobjects.ConnectionID) (aggregates.AccountLink, error)
}
