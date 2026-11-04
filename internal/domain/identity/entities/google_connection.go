// Package entities models identified children of identity aggregates.
package entities

import (
	"errors"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// GoogleConnection belongs to AccountLink; it is not an independent aggregate.
// Credentials, emails and persistence versions must never be stored here.
type GoogleConnection struct {
	subject valueobjects.GoogleAccountID
	scopes  valueobjects.OAuthScopes
	status  valueobjects.ConnectionStatus
}

func NewGoogleConnection(subject valueobjects.GoogleAccountID, scopes valueobjects.OAuthScopes) (GoogleConnection, error) {
	return RestoreGoogleConnection(subject, scopes, valueobjects.ConnectionActive)
}

func RestoreGoogleConnection(subject valueobjects.GoogleAccountID, scopes valueobjects.OAuthScopes, status valueobjects.ConnectionStatus) (GoogleConnection, error) {
	if !subject.Valid() || !scopes.Valid() || !status.Valid() {
		return GoogleConnection{}, errors.New("invalid Google connection")
	}
	return GoogleConnection{subject: subject, scopes: scopes, status: status}, nil
}

func (c GoogleConnection) AccountID() valueobjects.GoogleAccountID { return c.subject }
func (c GoogleConnection) Scopes() []string                        { return c.scopes.Values() }
func (c GoogleConnection) Status() valueobjects.ConnectionStatus   { return c.status }
func (c GoogleConnection) IsActive() bool                          { return c.status == valueobjects.ConnectionActive }

// Disconnect returns true only for the first state transition.
func (c *GoogleConnection) Disconnect() (bool, error) {
	if c == nil || !c.subject.Valid() || !c.scopes.Valid() || !c.status.Valid() {
		return false, errors.New("invalid Google connection")
	}
	if c.status == valueobjects.ConnectionDisconnected {
		return false, nil
	}
	c.status = valueobjects.ConnectionDisconnected
	return true, nil
}
