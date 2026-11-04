package workspace

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

const oauthSessionTTL = 10 * time.Minute

// LinkService is driven only by a trusted LINE identity boundary. Never
// accept OwnerID from public HTTP query parameters, form fields or headers.
type LinkService struct {
	provider    outbound.GoogleOAuthProvider
	sessions    outbound.OAuthSessionRepository
	connections outbound.OAuthConnectionRepository
	now         func() time.Time
}

type LinkedAccount struct {
	OwnerID       string
	GoogleSubject string
	VerifiedEmail string
}

func NewLinkService(provider outbound.GoogleOAuthProvider, sessions outbound.OAuthSessionRepository, connections outbound.OAuthConnectionRepository) (*LinkService, error) {
	if provider == nil || sessions == nil || connections == nil {
		return nil, errors.New("OAuth provider, state and connection repositories are required")
	}
	return &LinkService{provider: provider, sessions: sessions, connections: connections, now: time.Now}, nil
}

func secureNonce() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", errors.New("unable to generate OAuth nonce")
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// Begin receives a verified LINE owner from a separately authenticated
// entrypoint. A persisted, one-time state is required before URL exposure.
func (s *LinkService) Begin(ctx context.Context, verifiedOwner string, scopes []string) (string, error) {
	if ctx == nil || verifiedOwner == "" || verifiedOwner != strings.TrimSpace(verifiedOwner) || len(scopes) == 0 {
		return "", errors.New("verified owner and Google scopes required")
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
	url, err := s.provider.AuthorizationURL(state, verifier, nonce, scopes)
	if err != nil {
		return "", err
	}
	session := outbound.OAuthSession{
		OwnerID:   verifiedOwner,
		Verifier:  verifier,
		Nonce:     nonce,
		Scopes:    append([]string(nil), scopes...),
		ExpiresAt: s.now().UTC().Add(oauthSessionTTL),
	}
	if err := s.sessions.CreateSession(ctx, state, session); err != nil {
		return "", errors.New("unable to persist OAuth authorization state")
	}
	return url, nil
}

// Complete consumes the opaque state atomically before the external exchange.
// A failed exchange or persistence is terminal for this state: restart Begin.
func (s *LinkService) Complete(ctx context.Context, state, code string) (LinkedAccount, error) {
	if ctx == nil || len(state) != 43 || strings.TrimSpace(code) == "" {
		return LinkedAccount{}, errors.New("invalid OAuth callback")
	}
	if err := ctx.Err(); err != nil {
		return LinkedAccount{}, err
	}
	session, err := s.sessions.ConsumeSession(ctx, state)
	if err != nil {
		return LinkedAccount{}, errors.New("invalid or expired OAuth state")
	}
	if session.OwnerID == "" || session.Nonce == "" || !s.now().Before(session.ExpiresAt) {
		return LinkedAccount{}, errors.New("OAuth state expired")
	}
	identity, token, err := s.provider.ExchangeAndVerify(ctx, code, session.Verifier, session.Nonce)
	if err != nil {
		return LinkedAccount{}, errors.New("Google OAuth exchange failed")
	}
	if identity.Subject == "" || token.AccessToken == "" {
		return LinkedAccount{}, errors.New("Google identity or credentials missing")
	}
	confirmed := make(map[string]bool, len(identity.GrantedScopes))
	for _, granted := range identity.GrantedScopes {
		confirmed[granted] = true
	}
	for _, required := range session.Scopes {
		if required != "openid" && required != "email" && !confirmed[required] {
			return LinkedAccount{}, errors.New("Google authorization missing required scope")
		}
	}
	link := outbound.OAuthConnection{
		OwnerID:       session.OwnerID,
		GoogleSubject: identity.Subject,
		Scopes:        append([]string(nil), session.Scopes...),
		Credentials:   token,
		Active:        true,
	}
	if err := s.connections.SaveConnection(ctx, link); err != nil {
		return LinkedAccount{}, errors.New("unable to persist Google connection")
	}
	return LinkedAccount{OwnerID: session.OwnerID, GoogleSubject: identity.Subject, VerifiedEmail: identity.VerifiedEmail}, nil
}

// Disconnect first deactivates the local link so no new Workspace API calls
// can be authorized. Remote revocation is best effort; failures stay disabled.
func (s *LinkService) Disconnect(ctx context.Context, verifiedOwner, googleSubject string) error {
	if ctx == nil || verifiedOwner == "" || googleSubject == "" {
		return errors.New("verified owner and Google subject are required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	link, err := s.connections.GetConnection(ctx, verifiedOwner, googleSubject)
	if err != nil {
		return errors.New("Google connection unavailable")
	}
	if err := s.connections.DeactivateConnection(ctx, verifiedOwner, googleSubject); err != nil {
		return errors.New("unable to deactivate Google connection")
	}
	token := link.Credentials.RefreshToken
	if token == "" {
		token = link.Credentials.AccessToken
	}
	if token == "" {
		return nil
	}
	if err := s.provider.Revoke(ctx, token); err != nil {
		return errors.New("Google connection disabled locally; remote revocation remains unconfirmed")
	}
	return nil
}
