// Package factories converts verified identities or stored records into aggregates.
package factories

import (
	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// NewVerifiedLink requires upstream authentication and verified Google claims.
// The factory validates typed values but does not prove identity by itself.
func NewVerifiedLink(ownerID, googleSubject string, grantedScopes []string) (aggregates.AccountLink, error) {
	owner, err := valueobjects.NewLineUserID(ownerID)
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	subject, err := valueobjects.NewGoogleAccountID(googleSubject)
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	scopes, err := valueobjects.NewOAuthScopes(grantedScopes)
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	return aggregates.NewAccountLink(owner, subject, scopes)
}

// RestoreLink validates persisted state without claiming a new authorization.
func RestoreLink(ownerID, googleSubject string, grantedScopes []string, status valueobjects.ConnectionStatus) (aggregates.AccountLink, error) {
	owner, err := valueobjects.NewLineUserID(ownerID)
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	subject, err := valueobjects.NewGoogleAccountID(googleSubject)
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	scopes, err := valueobjects.NewOAuthScopes(grantedScopes)
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	return aggregates.RestoreAccountLink(owner, subject, scopes, status)
}
