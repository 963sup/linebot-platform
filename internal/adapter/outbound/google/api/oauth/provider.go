package oauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	infra "github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
)

// Provider adapts the Google-only official OAuth helper to application ports.
type Provider struct {
	flow           *infra.Flow
	http           *http.Client
	revokeEndpoint string
}

func NewProvider(flow *infra.Flow, httpClient *http.Client) (*Provider, error) {
	if flow == nil || httpClient == nil {
		return nil, errors.New("Google OAuth flow and HTTP client are required")
	}
	return &Provider{flow: flow, http: httpClient, revokeEndpoint: "https://oauth2.googleapis.com/revoke"}, nil
}

func (p *Provider) AuthorizationURL(state, verifier, nonce string, scopes []string) (string, error) {
	return p.flow.AuthorizationURL(state, verifier, nonce, scopes)
}

func (p *Provider) ExchangeAndVerify(ctx context.Context, code, verifier, nonce string) (outbound.GoogleIdentity, outbound.OAuthCredentials, error) {
	result, err := p.flow.ExchangeAndVerify(ctx, code, verifier, nonce)
	if err != nil {
		return outbound.GoogleIdentity{}, outbound.OAuthCredentials{}, err
	}
	return outbound.GoogleIdentity{Subject: result.User.Subject, VerifiedEmail: result.User.VerifiedEmail, GrantedScopes: result.GrantedScopes},
		outbound.OAuthCredentials{AccessToken: result.Token.AccessToken,
			RefreshToken: result.Token.RefreshToken, TokenType: result.Token.TokenType, Expiry: result.Token.Expiry}, nil
}

// Revoke must be called only after the local connection is disabled.
// Never expose the token, provider error body, or request URL in errors.
func (p *Provider) Revoke(ctx context.Context, token string) error {
	if p == nil || ctx == nil || strings.TrimSpace(token) == "" {
		return errors.New("Google OAuth token revocation requires authorized token")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.revokeEndpoint,
		strings.NewReader(url.Values{"token": {token}}.Encode()))
	if err != nil {
		return errors.New("cannot construct Google token revocation request")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.http.Do(request)
	if err != nil {
		return errors.New("Google token revocation result unknown")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusOK {
		return errors.New("Google token revocation failed or was not confirmed")
	}
	return nil
}

type oauthTokenFactory interface {
	TokenSource(context.Context, *oauth2.Token) (oauth2.TokenSource, error)
}

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
