// Package aggregates owns the consistency boundary of account linking.
package aggregates

import (
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/identity/entities"
	"github.com/963sup/linebot-platform/internal/domain/identity/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// AccountLink is the aggregate root for one LINE user and Google subject.
// OAuth credentials and Firestore metadata never belong to this model.
type AccountLink struct {
	owner   valueobjects.LineUserID
	google  entities.GoogleConnection
	pending []events.IdentityEvent
}

func NewAccountLink(owner valueobjects.LineUserID, subject valueobjects.GoogleAccountID, scopes valueobjects.OAuthScopes) (AccountLink, error) {
	google, err := entities.NewGoogleConnection(subject, scopes)
	if !owner.Valid() || err != nil {
		return AccountLink{}, errors.New("invalid account link")
	}
	id, err := valueobjects.NewConnectionID(owner, subject)
	if err != nil {
		return AccountLink{}, err
	}
	return AccountLink{
		owner:   owner,
		google:  google,
		pending: []events.IdentityEvent{events.AccountLinked{ID: id}},
	}, nil
}

// RestoreAccountLink rehydrates persisted state without raising new events.
func RestoreAccountLink(owner valueobjects.LineUserID, subject valueobjects.GoogleAccountID, scopes valueobjects.OAuthScopes, status valueobjects.ConnectionStatus) (AccountLink, error) {
	google, err := entities.RestoreGoogleConnection(subject, scopes, status)
	if !owner.Valid() || err != nil {
		return AccountLink{}, errors.New("invalid persisted account link")
	}
	return AccountLink{owner: owner, google: google}, nil
}

func (a AccountLink) Owner() valueobjects.LineUserID              { return a.owner }
func (a AccountLink) GoogleAccount() valueobjects.GoogleAccountID { return a.google.AccountID() }
func (a AccountLink) Scopes() []string                            { return a.google.Scopes() }
func (a AccountLink) Status() valueobjects.ConnectionStatus       { return a.google.Status() }
func (a AccountLink) IsActive() bool                              { return a.owner.Valid() && a.google.IsActive() }
func (a AccountLink) ConnectionID() (valueobjects.ConnectionID, error) {
	return valueobjects.NewConnectionID(a.owner, a.google.AccountID())
}

// PendingEvents returns a detached snapshot. Do not publish until persistence commits.
func (a AccountLink) PendingEvents() []events.IdentityEvent {
	return append([]events.IdentityEvent(nil), a.pending...)
}

// AcknowledgeEvents clears events after the aggregate state and event records
// have been committed successfully by the application boundary.
func (a *AccountLink) AcknowledgeEvents() {
	if a != nil {
		a.pending = nil
	}
}

// Disconnect is idempotent; only the first transition emits AccountUnlinked.
func (a *AccountLink) Disconnect() error {
	if a == nil || !a.owner.Valid() {
		return errors.New("invalid account link")
	}
	changed, err := a.google.Disconnect()
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	id, err := a.ConnectionID()
	if err != nil {
		return err
	}
	a.pending = append(a.pending, events.AccountUnlinked{ID: id})
	return nil
}
