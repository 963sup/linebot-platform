// Package aggregates defines identity aggregate invariants.
package aggregates

import (
	"errors"
	"github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// AccountLink relates a verified LINE user to a verified Google subject.
// It never contains OAuth credentials.
type AccountLink struct {
	owner   valueobjects.LineUserID
	subject valueobjects.GoogleAccountID
	scopes  valueobjects.OAuthScopes
	status  valueobjects.ConnectionStatus
}

func NewAccountLink(owner valueobjects.LineUserID, subject valueobjects.GoogleAccountID, scopes valueobjects.OAuthScopes) (AccountLink, error) {
	if !owner.Valid() || !subject.Valid() || !scopes.Valid() {
		return AccountLink{}, errors.New("invalid account link")
	}
	return AccountLink{owner: owner, subject: subject, scopes: scopes, status: valueobjects.ConnectionActive}, nil
}

// RestoreAccountLink validates persisted state without changing its lifecycle.
func RestoreAccountLink(owner valueobjects.LineUserID, subject valueobjects.GoogleAccountID, scopes valueobjects.OAuthScopes, status valueobjects.ConnectionStatus) (AccountLink, error) {
	if !owner.Valid() || !subject.Valid() || !scopes.Valid() || !status.Valid() {
		return AccountLink{}, errors.New("invalid persisted account link")
	}
	return AccountLink{owner: owner, subject: subject, scopes: scopes, status: status}, nil
}
func (a AccountLink) Owner() valueobjects.LineUserID              { return a.owner }
func (a AccountLink) GoogleAccount() valueobjects.GoogleAccountID { return a.subject }
func (a AccountLink) Scopes() []string                            { return a.scopes.Values() }
func (a AccountLink) Status() valueobjects.ConnectionStatus       { return a.status }
func (a AccountLink) IsActive() bool                              { return a.status == valueobjects.ConnectionActive }

// Disconnect is idempotent; reactivation requires a new verified authorization flow.
func (a *AccountLink) Disconnect() error {
	if a == nil || !a.owner.Valid() || !a.subject.Valid() {
		return errors.New("invalid account link")
	}
	a.status = valueobjects.ConnectionDisconnected
	return nil
}
