package identity

import (
	"context"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
)

// OAuthSession is a one-time binding between an authenticated LINE principal
// and an OAuth authorization attempt. Adapters own its storage representation.
type OAuthSession struct {
	OwnerID   string
	Verifier  string
	Nonce     string
	Scopes    []string
	ExpiresAt time.Time
}

// OAuthCredentials are application data required by an OAuth adapter. They
// deliberately stay outside the Identity aggregate.
type OAuthCredentials struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
}

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

// OAuthProvider is consumer-owned; Google SDK types must be mapped by adapters.
type OAuthProvider interface {
	AuthorizationURL(state, verifier, nonce string, scopes []string) (string, error)
	ExchangeAndVerify(context.Context, string, string, string) (GoogleIdentity, OAuthCredentials, error)
	Revoke(context.Context, string) error
}

type OAuthSessionRepository interface {
	CreateSession(context.Context, string, OAuthSession) error
	ConsumeSession(context.Context, string) (OAuthSession, error)
}

type ConnectionStore interface {
	SaveConnection(context.Context, OAuthAuthorization) error
	GetConnection(context.Context, string, string) (OAuthAuthorization, error)
	DeactivateConnection(context.Context, OAuthAuthorization) error
}
