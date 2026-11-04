package outbound

import (
	"context"
	"time"
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

type OAuthConnection struct {
	OwnerID       string
	GoogleSubject string
	Scopes        []string
	Credentials   OAuthCredentials
	Version       int64
	Active        bool
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

type OAuthConnectionRepository interface {
	SaveConnection(context.Context, OAuthConnection) error
	GetConnection(context.Context, string, string) (OAuthConnection, error)
	UpdateCredentials(context.Context, OAuthConnection, int64) error
	DeactivateConnection(context.Context, string, string) error
}
