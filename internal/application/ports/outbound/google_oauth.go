package outbound

import (
	"context"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
)

// OAuthSession is a one-time, short-lived binding between an already
// authenticated LINE principal and a Google authorization flow.
type OAuthSession struct {
	OwnerID   string
	Verifier  string
	Nonce     string
	Scopes    []string
	ExpiresAt time.Time
}

type OAuthCredentials struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
}

// OAuthAuthorization couples the domain aggregate to technical credentials.
// Identity, scopes and lifecycle state exist only in Link, never duplicated here.
type OAuthAuthorization struct {
	Link        aggregates.AccountLink
	Credentials OAuthCredentials
	Version     int64
}

type GoogleIdentity struct {
	Subject       string
	VerifiedEmail string
	GrantedScopes []string
}

type GoogleOAuthProvider interface {
	AuthorizationURL(state, verifier, nonce string, scopes []string) (string, error)
	ExchangeAndVerify(context.Context, string, string, string) (GoogleIdentity, OAuthCredentials, error)
	Revoke(context.Context, string) error
}

type OAuthSessionRepository interface {
	CreateSession(context.Context, string, OAuthSession) error
	ConsumeSession(context.Context, string) (OAuthSession, error)
}

