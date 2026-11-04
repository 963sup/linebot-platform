package oauth

import (
	"context"
	"errors"
	"time"

	"golang.org/x/oauth2"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type oauthTokenFactory interface {
	TokenSource(context.Context, *oauth2.Token) (oauth2.TokenSource, error)
}

// PersistedTokenSource refreshes a user OAuth token and persists updates with
// the connection version as a compare-and-swap guard.
type PersistedTokenSource struct {
	ctx     context.Context
	flow    oauthTokenFactory
	store   authorizationTokenStore
	owner   string
	subject string
}

// authorizationTokenStore is private to the refresh consumer, not a global four-method port.
type authorizationTokenStore interface {
	GetConnection(context.Context, string, string) (outbound.OAuthAuthorization, error)
	UpdateCredentials(context.Context, outbound.OAuthAuthorization, int64) error
}

func NewPersistedTokenSource(ctx context.Context, flow oauthTokenFactory, store authorizationTokenStore, owner, subject string) (*PersistedTokenSource, error) {
	if ctx == nil || flow == nil || store == nil || owner == "" || subject == "" {
		return nil, errors.New("linked Google account identity, store and flow required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &PersistedTokenSource{ctx: ctx, flow: flow, store: store, owner: owner, subject: subject}, nil
}

func (p *PersistedTokenSource) Token() (*oauth2.Token, error) {
	if p == nil {
		return nil, errors.New("OAuth TokenSource not configured")
	}
	if err := p.ctx.Err(); err != nil {
		return nil, err
	}
	current, err := p.store.GetConnection(p.ctx, p.owner, p.subject)
	if err != nil {
		return nil, errors.New("Google account not linked")
	}
	if !current.Link.IsActive() {
		return nil, errors.New("Google account disabled")
	}
	original := &oauth2.Token{
		AccessToken:  current.Credentials.AccessToken,
		RefreshToken: current.Credentials.RefreshToken,
		TokenType:    current.Credentials.TokenType,
		Expiry:       current.Credentials.Expiry,
	}
	source, err := p.flow.TokenSource(p.ctx, original)
	if err != nil {
		return nil, err
	}
	updated, err := source.Token()
	if err != nil {
		return nil, errors.New("Google OAuth token refresh failed")
	}
	if updated.AccessToken == original.AccessToken && updated.Expiry.Equal(original.Expiry) {
		return updated, nil
	}
	next := current
	next.Credentials = outbound.OAuthCredentials{
		AccessToken:  updated.AccessToken,
		RefreshToken: updated.RefreshToken,
		TokenType:    updated.TokenType,
		Expiry:       updated.Expiry,
	}
	if next.Credentials.RefreshToken == "" {
		next.Credentials.RefreshToken = original.RefreshToken
	}
	if err := p.store.UpdateCredentials(p.ctx, next, current.Version); err != nil {
		// After another replica refreshes the token, a stale writer MUST NOT
		// overwrite it. Reload the newest version; if unavailable fail closed.
		latest, readErr := p.store.GetConnection(p.ctx, p.owner, p.subject)
		if readErr != nil || !latest.Link.IsActive() || latest.Version <= current.Version ||
			latest.Credentials.AccessToken == "" || !latest.Credentials.Expiry.After(time.Now().Add(30*time.Second)) {
			return nil, errors.New("Google OAuth token refresh persistence uncertain")
		}
		return &oauth2.Token{
			AccessToken:  latest.Credentials.AccessToken,
			RefreshToken: latest.Credentials.RefreshToken,
			TokenType:    latest.Credentials.TokenType,
			Expiry:       latest.Credentials.Expiry,
		}, nil
	}
	return updated, nil
}
