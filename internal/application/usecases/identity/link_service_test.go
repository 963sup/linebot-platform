package identity

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type oauthProviderFake struct {
	mu         sync.Mutex
	state      string
	verifier   string
	nonce      string
	exchanges  int
	grants     []string
	revokeErr  error
	revokeWith string
}

func (f *oauthProviderFake) AuthorizationURL(state, verifier, nonce string, _ []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state, f.verifier, f.nonce = state, verifier, nonce
	return "https://accounts.example/authorize?state=" + state, nil
}

func (f *oauthProviderFake) ExchangeAndVerify(_ context.Context, code, verifier, nonce string) (GoogleIdentity, OAuthCredentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchanges++
	if code == "" || verifier != f.verifier || nonce != f.nonce {
		return GoogleIdentity{}, OAuthCredentials{}, errors.New("invalid exchange")
	}
	return GoogleIdentity{Subject: "google-subject", VerifiedEmail: "user@example.com", GrantedScopes: f.grants},
		OAuthCredentials{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil
}

func (f *oauthProviderFake) Revoke(_ context.Context, token string) error {
	f.revokeWith = token
	return f.revokeErr
}

type oauthStoreFake struct {
	sessions map[string]OAuthSession
	links    map[string]OAuthAuthorization
}

func newOAuthStoreFake() *oauthStoreFake {
	return &oauthStoreFake{sessions: map[string]OAuthSession{}, links: map[string]OAuthAuthorization{}}
}

func (f *oauthStoreFake) CreateSession(_ context.Context, state string, session OAuthSession) error {
	if _, exists := f.sessions[state]; exists {
		return errors.New("duplicate state")
	}
	f.sessions[state] = session
	return nil
}

func (f *oauthStoreFake) ConsumeSession(_ context.Context, state string) (OAuthSession, error) {
	session, exists := f.sessions[state]
	if !exists {
		return OAuthSession{}, errors.New("state not found")
	}
	delete(f.sessions, state)
	return session, nil
}

func (f *oauthStoreFake) SaveConnection(_ context.Context, authorization OAuthAuthorization) error {
	key := authorization.Link.Owner().String() + "|" + authorization.Link.GoogleAccount().String()
	f.links[key] = authorization
	return nil
}

func (f *oauthStoreFake) GetConnection(_ context.Context, owner, subject string) (OAuthAuthorization, error) {
	authorization, exists := f.links[owner+"|"+subject]
	if !exists || !authorization.Link.IsActive() {
		return OAuthAuthorization{}, errors.New("connection not found")
	}
	return authorization, nil
}

func (f *oauthStoreFake) DeactivateConnection(_ context.Context, owner, subject string) error {
	key := owner + "|" + subject
	authorization, exists := f.links[key]
	if !exists {
		return errors.New("connection not found")
	}
	if err := authorization.Link.Disconnect(); err != nil {
		return err
	}
	authorization.Credentials = OAuthCredentials{}
	f.links[key] = authorization
	return nil
}

func TestLinkServiceConsumesStateOnceAndBindsScopes(t *testing.T) {
	t.Parallel()
	scope := "https://www.googleapis.com/auth/calendar.events"
	provider := &oauthProviderFake{grants: []string{scope}}
	store := newOAuthStoreFake()
	service, err := NewLinkService(provider, store, store)
	if err != nil {
		t.Fatal(err)
	}
	url, err := service.Begin(context.Background(), "line-owner", []string{scope})
	if err != nil || url == "" {
		t.Fatalf("Begin() = (%q, %v)", url, err)
	}
	account, err := service.Complete(context.Background(), provider.state, "code")
	if err != nil {
		t.Fatal(err)
	}
	if account.OwnerID != "line-owner" || account.GoogleSubject != "google-subject" || account.VerifiedEmail == "" {
		t.Fatalf("Complete() = %#v", account)
	}
	if _, err := service.Complete(context.Background(), provider.state, "code"); err == nil {
		t.Fatal("replayed OAuth state accepted")
	}
	if provider.exchanges != 1 {
		t.Fatalf("exchange calls = %d, want 1", provider.exchanges)
	}
}

func TestLinkServiceRejectsInvalidInputAndMissingScope(t *testing.T) {
	t.Parallel()
	provider := &oauthProviderFake{grants: []string{"openid"}}
	store := newOAuthStoreFake()
	service, _ := NewLinkService(provider, store, store)
	if _, err := service.Begin(context.Background(), "", []string{"openid"}); err == nil {
		t.Fatal("empty owner accepted")
	}
	if _, err := service.Begin(context.Background(), "line-owner", []string{"invalid scope"}); err == nil {
		t.Fatal("invalid scope accepted")
	}
	requested := "https://www.googleapis.com/auth/forms.body"
	if _, err := service.Begin(context.Background(), "line-owner", []string{requested}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(context.Background(), provider.state, "code"); err == nil {
		t.Fatal("missing granted scope accepted")
	}
	if len(store.links) != 0 {
		t.Fatal("unauthorized connection persisted")
	}
}

func TestDisconnectFailsClosedBeforeRemoteRevocation(t *testing.T) {
	t.Parallel()
	provider := &oauthProviderFake{grants: []string{"openid"}, revokeErr: errors.New("network failure")}
	store := newOAuthStoreFake()
	service, _ := NewLinkService(provider, store, store)
	if _, err := service.Begin(context.Background(), "line-owner", []string{"openid"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(context.Background(), provider.state, "code"); err != nil {
		t.Fatal(err)
	}
	if err := service.Disconnect(context.Background(), "line-owner", "google-subject"); err == nil {
		t.Fatal("remote revocation uncertainty not reported")
	}
	if _, err := store.GetConnection(context.Background(), "line-owner", "google-subject"); err == nil {
		t.Fatal("connection remained active after disconnect")
	}
	if provider.revokeWith != "refresh" {
		t.Fatalf("revoked token = %q, want refresh", provider.revokeWith)
	}
}
