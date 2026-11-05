// Package identity orchestrates account linking without depending on OAuth SDKs.
package identity

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/policies"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

const oauthSessionTTL = 10 * time.Minute

var (
	ErrInvalidOAuthRequest         = errors.New("invalid OAuth request")
	ErrInvalidOAuthState           = errors.New("invalid or expired OAuth state")
	ErrOAuthProviderUnavailable    = errors.New("OAuth provider unavailable")
	ErrOAuthPersistenceUnavailable = errors.New("OAuth persistence unavailable")
	ErrOAuthRevocationUnknown      = errors.New("OAuth revocation outcome unknown")
)

func contextOr(ctx context.Context, fallback error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fallback
}

type LinkService struct {
	provider    OAuthProvider
	sessions    OAuthSessionRepository
	connections ConnectionStore
	now         func() time.Time
}

type LinkedAccount struct {
	OwnerID       string
	GoogleSubject string
	VerifiedEmail string
}

func NewLinkService(provider OAuthProvider, sessions OAuthSessionRepository, connections ConnectionStore) (*LinkService, error) {
	if provider == nil || sessions == nil || connections == nil {
		return nil, errors.New("OAuth provider, state and connection repositories are required")
	}
	return &LinkService{provider: provider, sessions: sessions, connections: connections, now: time.Now}, nil
}

func secureNonce() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", errors.New("unable to generate OAuth nonce")
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

// Begin accepts only an identity already authenticated by an inbound boundary.
func (s *LinkService) Begin(ctx context.Context, verifiedOwner string, scopes []string) (string, error) {
	if ctx == nil {
		return "", ErrInvalidOAuthRequest
	}
	owner, err := valueobjects.NewLineUserID(verifiedOwner)
	if err != nil {
		return "", ErrInvalidOAuthRequest
	}
	requestedScopes, err := valueobjects.NewOAuthScopes(scopes)
	if err != nil {
		return "", ErrInvalidOAuthRequest
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	state, err := secureNonce()
	if err != nil {
		return "", err
	}
	verifier, err := secureNonce()
	if err != nil {
		return "", err
	}
	nonce, err := secureNonce()
	if err != nil {
		return "", err
	}
	url, err := s.provider.AuthorizationURL(state, verifier, nonce, requestedScopes.Values())
	if err != nil {
		return "", ErrOAuthProviderUnavailable
	}
	session := OAuthSession{
		OwnerID: owner.String(), Verifier: verifier, Nonce: nonce,
		Scopes: requestedScopes.Values(), ExpiresAt: s.now().UTC().Add(oauthSessionTTL),
	}
	if err := s.sessions.CreateSession(ctx, state, session); err != nil {
		return "", contextOr(ctx, ErrOAuthPersistenceUnavailable)
	}
	return url, nil
}

// Complete consumes state before token exchange, making callback replay fail closed.
func (s *LinkService) Complete(ctx context.Context, state, code string) (LinkedAccount, error) {
	if ctx == nil || len(state) != 43 || strings.TrimSpace(code) == "" {
		return LinkedAccount{}, ErrInvalidOAuthRequest
	}
	if err := ctx.Err(); err != nil {
		return LinkedAccount{}, err
	}
	session, err := s.sessions.ConsumeSession(ctx, state)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return LinkedAccount{}, ctxErr
		}
		if errors.Is(err, ErrOAuthSessionNotFound) {
			return LinkedAccount{}, ErrInvalidOAuthState
		}
		return LinkedAccount{}, ErrOAuthPersistenceUnavailable
	}
	if session.OwnerID == "" || session.Nonce == "" || !s.now().Before(session.ExpiresAt) {
		return LinkedAccount{}, ErrInvalidOAuthState
	}
	googleIdentity, credentials, err := s.provider.ExchangeAndVerify(ctx, code, session.Verifier, session.Nonce)
	if err != nil {
		return LinkedAccount{}, contextOr(ctx, ErrOAuthProviderUnavailable)
	}
	if googleIdentity.Subject == "" || credentials.AccessToken == "" {
		return LinkedAccount{}, errors.New("Google identity or credentials missing")
	}
	requestedScopes, err := valueobjects.NewOAuthScopes(session.Scopes)
	if err != nil || policies.ValidateGrantedScopes(requestedScopes, googleIdentity.GrantedScopes) != nil {
		return LinkedAccount{}, errors.New("Google authorization missing required scope")
	}
	accountLink, err := factories.NewVerifiedLink(session.OwnerID, googleIdentity.Subject, session.Scopes)
	if err != nil {
		return LinkedAccount{}, errors.New("invalid verified account link")
	}
	if err := s.connections.SaveConnection(ctx, OAuthAuthorization{Link: accountLink, Credentials: credentials}); err != nil {
		return LinkedAccount{}, contextOr(ctx, ErrOAuthPersistenceUnavailable)
	}
	accountLink.AcknowledgeEvents()
	return LinkedAccount{
		OwnerID: session.OwnerID, GoogleSubject: googleIdentity.Subject, VerifiedEmail: googleIdentity.VerifiedEmail,
	}, nil
}

// Disconnect disables local authorization before best-effort remote revocation.
func (s *LinkService) Disconnect(ctx context.Context, verifiedOwner, googleSubject string) error {
	if ctx == nil || verifiedOwner == "" || googleSubject == "" {
		return ErrInvalidOAuthRequest
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	authorization, err := s.connections.GetConnection(ctx, verifiedOwner, googleSubject)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if errors.Is(err, ErrOAuthConnectionNotFound) {
			return ErrInvalidOAuthState
		}
		return ErrOAuthPersistenceUnavailable
	}
	owner, err := valueobjects.NewLineUserID(verifiedOwner)
	if err != nil {
		return errors.New("invalid LINE account identity")
	}
	subject, err := valueobjects.NewGoogleAccountID(googleSubject)
	if err != nil || policies.ValidateUnlink(authorization.Link, owner, subject) != nil {
		return errors.New("invalid Google connection state")
	}
	if err := authorization.Link.Disconnect(); err != nil {
		return err
	}
	if err := s.connections.DeactivateConnection(ctx, authorization); err != nil {
		return contextOr(ctx, ErrOAuthPersistenceUnavailable)
	}
	authorization.Link.AcknowledgeEvents()
	token := authorization.Credentials.RefreshToken
	if token == "" {
		token = authorization.Credentials.AccessToken
	}
	if token == "" {
		return nil
	}
	if err := s.provider.Revoke(ctx, token); err != nil {
		return contextOr(ctx, ErrOAuthRevocationUnknown)
	}
	return nil
}
