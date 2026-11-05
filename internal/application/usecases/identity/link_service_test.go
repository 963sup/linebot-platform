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
	exchange   func(context.Context) error
}

func (f *oauthProviderFake) AuthorizationURL(state, verifier, nonce string, _ []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state, f.verifier, f.nonce = state, verifier, nonce
	return "https://accounts.example/authorize?state=" + state, nil
}

func (f *oauthProviderFake) ExchangeAndVerify(ctx context.Context, code, verifier, nonce string) (GoogleIdentity, OAuthCredentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchanges++
	if f.exchange != nil {
		if err := f.exchange(ctx); err != nil {
			return GoogleIdentity{}, OAuthCredentials{}, err
		}
	}
	if code == "" || verifier != f.verifier || nonce != f.nonce {
		return GoogleIdentity{}, OAuthCredentials{}, errors.New("invalid exchange")
	}
	return GoogleIdentity{Subject: "google-subject", VerifiedEmail: "user@example.com", GrantedScopes: f.grants},
		OAuthCredentials{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil
}

func TestLinkServicePreservesCancellationFromProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &oauthProviderFake{grants: []string{"openid"}}
	store := newOAuthStoreFake()
	service, _ := NewLinkService(provider, store, store)
	if _, err := service.Begin(ctx, "line-owner", []string{"openid"}); err != nil {
		t.Fatal(err)
	}
	provider.exchange = func(context.Context) error {
		cancel()
		return errors.New("transport stopped")
	}
	if _, err := service.Complete(ctx, provider.state, "code"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Complete() error = %v, want context.Canceled", err)
	}
}

func (f *oauthProviderFake) Revoke(_ context.Context, token string) error {
	f.revokeWith = token
	return f.revokeErr
}

type oauthStoreFake struct {
	sessions   map[string]OAuthSession
	links      map[string]OAuthAuthorization
	consumeErr error
	getErr     error
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
	if f.consumeErr != nil {
		return OAuthSession{}, f.consumeErr
	}
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
	if f.getErr != nil {
		return OAuthAuthorization{}, f.getErr
	}
	authorization, exists := f.links[owner+"|"+subject]
	if !exists || !authorization.Link.IsActive() {
		return OAuthAuthorization{}, errors.New("connection not found")
	}
	return authorization, nil
}

func TestLinkServiceClassifiesSessionRepositoryFailures(t *testing.T) {
	provider := &oauthProviderFake{grants: []string{"openid"}}
	for _, test := range []struct {
		name          string
		repositoryErr error
		want          error
	}{
		{name: "missing", repositoryErr: ErrOAuthSessionNotFound, want: ErrInvalidOAuthState},
		{name: "outage", repositoryErr: errors.New("firestore unavailable"), want: ErrOAuthPersistenceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newOAuthStoreFake()
			service, _ := NewLinkService(provider, store, store)
			if _, err := service.Begin(context.Background(), "line-owner", []string{"openid"}); err != nil {
				t.Fatal(err)
			}
			store.consumeErr = test.repositoryErr
			if _, err := service.Complete(context.Background(), provider.state, "code"); !errors.Is(err, test.want) {
				t.Fatalf("Complete() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestLinkServiceClassifiesConnectionRepositoryFailures(t *testing.T) {
	provider := &oauthProviderFake{}
	for _, test := range []struct {
		name          string
		repositoryErr error
		want          error
	}{
		{name: "missing", repositoryErr: ErrOAuthConnectionNotFound, want: ErrInvalidOAuthState},
		{name: "outage", repositoryErr: errors.New("firestore unavailable"), want: ErrOAuthPersistenceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newOAuthStoreFake()
			store.getErr = test.repositoryErr
			service, _ := NewLinkService(provider, store, store)
			if err := service.Disconnect(context.Background(), "line-owner", "google-subject"); !errors.Is(err, test.want) {
				t.Fatalf("Disconnect() error = %v, want %v", err, test.want)
			}
		})
	}
}

func (f *oauthStoreFake) DeactivateConnection(_ context.Context, authorization OAuthAuthorization) error {
	key := authorization.Link.Owner().String() + "|" + authorization.Link.GoogleAccount().String()
	stored, exists := f.links[key]
	if !exists {
		return errors.New("connection not found")
	}
	stored.Link = authorization.Link
	stored.Credentials = OAuthCredentials{}
	f.links[key] = stored
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
	if err := service.Disconnect(context.Background(), "line-owner", "google-subject"); !errors.Is(err, ErrOAuthRevocationUnknown) {
		t.Fatalf("Disconnect() error = %v, want ErrOAuthRevocationUnknown", err)
	}
	if _, err := store.GetConnection(context.Background(), "line-owner", "google-subject"); err == nil {
		t.Fatal("connection remained active after disconnect")
	}
	if provider.revokeWith != "refresh" {
		t.Fatalf("revoked token = %q, want refresh", provider.revokeWith)
	}
}
