package oauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	identity "github.com/963sup/linebot-platform/internal/application/usecases/identity"
	infra "github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
)

// Provider adapts the Google-only official OAuth helper to application ports.
type Provider struct {
	flow           *infra.Flow
	http           *http.Client
	revokeEndpoint string
}

var _ identity.OAuthProvider = (*Provider)(nil)

func NewProvider(flow *infra.Flow, httpClient *http.Client) (*Provider, error) {
	if flow == nil || httpClient == nil {
		return nil, errors.New("Google OAuth flow and HTTP client are required")
	}
	return &Provider{flow: flow, http: httpClient, revokeEndpoint: "https://oauth2.googleapis.com/revoke"}, nil
}

func (p *Provider) AuthorizationURL(state, verifier, nonce string, scopes []string) (string, error) {
	return p.flow.AuthorizationURL(state, verifier, nonce, scopes)
}

func (p *Provider) ExchangeAndVerify(ctx context.Context, code, verifier, nonce string) (identity.GoogleIdentity, identity.OAuthCredentials, error) {
	result, err := p.flow.ExchangeAndVerify(ctx, code, verifier, nonce)
	if err != nil {
		return identity.GoogleIdentity{}, identity.OAuthCredentials{}, err
	}
	return identity.GoogleIdentity{Subject: result.User.Subject, VerifiedEmail: result.User.VerifiedEmail, GrantedScopes: result.GrantedScopes},
		identity.OAuthCredentials{AccessToken: result.Token.AccessToken,
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
