package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	identity "github.com/963sup/linebot-platform/internal/application/usecases/identity"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"golang.org/x/oauth2"
)

type fakeTokenSourceFactory struct {
	next *oauth2.Token
}

func (f fakeTokenSourceFactory) TokenSource(context.Context, *oauth2.Token) (oauth2.TokenSource, error) {
	return oauth2.StaticTokenSource(f.next), nil
}

type memoryConnectionStore struct {
	mu     sync.Mutex
	value  identity.OAuthAuthorization
	closed bool
}

func (s *memoryConnectionStore) SaveConnection(_ context.Context, v identity.OAuthAuthorization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = v
	return nil
}

func (s *memoryConnectionStore) GetConnection(_ context.Context, owner, subject string) (identity.OAuthAuthorization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.value.Link.IsActive() || s.value.Link.Owner().String() != owner || s.value.Link.GoogleAccount().String() != subject {
		return identity.OAuthAuthorization{}, errors.New("no connection")
	}
	return s.value, nil
}

func (s *memoryConnectionStore) UpdateCredentials(_ context.Context, v identity.OAuthAuthorization, version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || version != s.value.Version {
		return errors.New("version conflict")
	}
	v.Version = version + 1
	s.value = v
	return nil
}

func (s *memoryConnectionStore) DeactivateConnection(_ context.Context, _ identity.OAuthAuthorization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func makeStoredAccount(t *testing.T) *memoryConnectionStore {
	t.Helper()
	link, err := factories.NewVerifiedLink("verified-line", "verified-google", []string{
		"openid", "https://www.googleapis.com/auth/tasks.readonly",
	})
	if err != nil {
		t.Fatal(err)
	}
	return &memoryConnectionStore{value: identity.OAuthAuthorization{
		Link: link, Version: 7,
		Credentials: identity.OAuthCredentials{
			AccessToken: "expired-access", RefreshToken: "long-lived-refresh",
			TokenType: "Bearer", Expiry: time.Now().Add(-time.Hour),
		},
	}}
}

func TestPersistedTokenSourceRefreshAndCAS(t *testing.T) {
	store := makeStoredAccount(t)
	next := &oauth2.Token{AccessToken: "new-access", Expiry: time.Now().Add(time.Hour), TokenType: "Bearer"}
	source, err := NewPersistedTokenSource(context.Background(), fakeTokenSourceFactory{next: next}, store, "verified-line", "verified-google")
	if err != nil {
		t.Fatal(err)
	}
	token, err := source.Token()
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "new-access" {
		t.Fatal("refreshed access token not returned")
	}
	saved, err := store.GetConnection(context.Background(), "verified-line", "verified-google")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 8 || saved.Credentials.RefreshToken != "long-lived-refresh" {
		t.Fatal("CAS update lost refresh credential or version")
	}
	if _, err := NewPersistedTokenSource(context.Background(), fakeTokenSourceFactory{}, store, "", "verified-google"); err == nil {
		t.Fatal("anonymous owner accepted")
	}
	if _, err := NewPersistedTokenSource(context.Background(), fakeTokenSourceFactory{}, store, "verified-line", "other-google"); err != nil {
		t.Fatal("valid configuration rejected")
	}
	blocked, _ := NewPersistedTokenSource(context.Background(), fakeTokenSourceFactory{next: next}, store, "verified-line", "other-google")
	if _, err := blocked.Token(); err == nil {
		t.Fatal("cross-account token lookup succeeded")
	}
	if err := store.DeactivateConnection(context.Background(), store.value); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Token(); err == nil {
		t.Fatal("deactivated account allowed access")
	}
}

func TestRevokeProviderDoesNotLeakCredentials(t *testing.T) {
	var gotMethod, gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.URL.Query().Get("token") != "" {
			t.Fatal("token leaked to URL")
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		gotToken = r.PostForm.Get("token")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("provider-secret-error"))
	}))
	defer server.Close()
	provider := &Provider{http: server.Client(), revokeEndpoint: server.URL}
	err := provider.Revoke(context.Background(), "sensitive-token")
	if err == nil {
		t.Fatal("failed revocation reported as success")
	}
	if gotToken != "sensitive-token" || gotMethod != http.MethodPost {
		t.Fatal("incorrect revocation transport")
	}
	if strings.Contains(err.Error(), "provider-secret") || strings.Contains(err.Error(), "sensitive-token") {
		t.Fatal("sensitive token or provider error leaked")
	}
}
