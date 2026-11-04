package firestore

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// accountLinkFromConnection maps the existing technical persistence DTO into
// the domain model without leaking OAuth credentials or Firestore metadata.
// The legacy wire/storage representation stays unchanged during migration.
func accountLinkFromConnection(conn outbound.OAuthConnection) (aggregates.AccountLink, error) {
	status := valueobjects.ConnectionDisconnected
	if conn.Active {
		status = valueobjects.ConnectionActive
	}
	return factories.RestoreLink(conn.OwnerID, conn.GoogleSubject, conn.Scopes, status)
}

// FindConnection implements the domain's read-only repository contract.
// Legacy credential reads still fail closed on disabled records.
func (s *OAuthStore) FindConnection(ctx context.Context, id valueobjects.ConnectionID) (aggregates.AccountLink, error) {
	if !id.Valid() {
		return aggregates.AccountLink{}, errors.New("invalid connection ID")
	}
	conn, err := s.GetConnection(ctx, id.Owner().String(), id.GoogleAccount().String())
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	return accountLinkFromConnection(conn)
}

var _ repositories.ConnectionReader = (*OAuthStore)(nil)
