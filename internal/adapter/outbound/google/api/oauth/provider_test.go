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

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"golang.org/x/oauth2"
)

type fakeTokenSourceFactory struct {
	next *oauth2.Token
}

func (f fakeTokenSourceFactory) TokenSource(context.Context, *oauth2.Token) (oauth2.TokenSource, error) {
	return oauth2.StaticTokenSource(f.next), nil
}

type memoryConnectionStore struct {
	mu sync.Mutex
	value outbound.OAuthConnection
	closed bool
}

func (s *memoryConnectionStore) SaveConnection(_ context.Context, v outbound.OAuthConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = v
	return nil
}

func (s *memoryConnectionStore) GetConnection(_ context.Context, owner, subject string) (outbound.OAuthConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.value.Active || s.value.OwnerID != owner || s.value.GoogleSubject != subject {
		return outbound.OAuthConnection{}, errors.New("no connection")
	}
	return s.value, nil
}

func (s *memoryConnectionStore) UpdateCredentials(_ context.Context, v outbound.OAuthConnection, version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || version != s.value.Version {
		return errors.New("version conflict")
	}
	v.Version = version + 1
	s.value = v
	return nil
}

func (s *memoryConnectionStore) DeactivateConnection(_ context.Context, _, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func makeStoredAccount() *memoryConnectionStore {
	return &memoryConnectionStore{value: outbound.OAuthConnection{
		OwnerID: "verified-line", GoogleSubject: "verified-google",
		Active: true, Version: 7,
		Scopes: []string{"openid", "https://www.googleapis.com/auth/tasks.readonly"},
		Credentials: outbound.OAuthCredentials{
			AccessToken: "expired-access", RefreshToken: "long-lived-refresh",
			TokenType: "Bearer", Expiry: time.Now().Add(-time.Hour),
		},
	}}
}

func TestPersistedTokenSourceRefreshAndCAS(t *testing.T) {
	store := makeStoredAccount()
	next := &oauth2.Token{AccessToken: "new-access", Expiry: time.Now().Add(time.Hour), TokenType: "Bearer"}
	source, err := NewPersistedTokenSource(context.Background(), fakeTokenSourceFactory{next: next}, store, "verified-line", "verified-google")
	if err != nil { t.Fatal(err) }
	token, err := source.Token()
	if err != nil { t.Fatal(err) }
	if token.AccessToken != "new-access" { t.Fatal("refreshed access token not returned") }
	saved, err := store.GetConnection(context.Background(), "verified-line", "verified-google")
	if err != nil { t.Fatal(err) }
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
	if _, err := blocked.Token(); err == nil { t.Fatal("cross-account token lookup succeeded") }
	if err := store.DeactivateConnection(context.Background(), "verified-line", "verified-google"); err != nil { t.Fatal(err) }
	if _, err := source.Token(); err == nil { t.Fatal("deactivated account allowed access") }
}

func TestRevokeProviderDoesNotLeakCredentials(t *testing.T) {
	var gotMethod, gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.URL.Query().Get("token") != "" { t.Fatal("token leaked to URL") }
		if err := r.ParseForm(); err != nil { t.Error(err) }
		gotToken = r.PostForm.Get("token")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("provider-secret-error"))
	}))
	defer server.Close()
	provider := &Provider{http: server.Client(), revokeEndpoint: server.URL}
	err := provider.Revoke(context.Background(), "sensitive-token")
	if err == nil { t.Fatal("failed revocation reported as success") }
	if gotToken != "sensitive-token" || gotMethod != http.MethodPost { t.Fatal("incorrect revocation transport") }
	if strings.Contains(err.Error(), "provider-secret") || strings.Contains(err.Error(), "sensitive-token") {
		t.Fatal("sensitive token or provider error leaked")
	}
}
