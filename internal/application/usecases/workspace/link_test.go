package workspace

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeGoogleProvider struct {
	mu        sync.Mutex
	state     string
	verifier  string
	nonce     string
	exchanges int
	revokeErr error
}

func (f *fakeGoogleProvider) AuthorizationURL(state, verifier, nonce string, scopes []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state, f.verifier, f.nonce = state, verifier, nonce
	return "https://accounts.google.com/oauth?state=" + state, nil
}
func (f *fakeGoogleProvider) ExchangeAndVerify(_ context.Context, code, verifier, nonce string) (outbound.GoogleIdentity, outbound.OAuthCredentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchanges++
	if code == "" || verifier != f.verifier || nonce != f.nonce {
		return outbound.GoogleIdentity{}, outbound.OAuthCredentials{}, errors.New("invalid")
	}
	return outbound.GoogleIdentity{Subject: "google-sub", GrantedScopes: []string{"https://www.googleapis.com/auth/forms.body"}}, outbound.OAuthCredentials{AccessToken: "secret-access", RefreshToken: "secret-refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil
}
func (f *fakeGoogleProvider) Revoke(context.Context, string) error { return f.revokeErr }

type fakeOAuthStore struct {
	mu       sync.Mutex
	sessions map[string]outbound.OAuthSession
	links    map[string]outbound.OAuthConnection
}

func newFakeOAuthStore() *fakeOAuthStore {
	return &fakeOAuthStore{sessions: map[string]outbound.OAuthSession{}, links: map[string]outbound.OAuthConnection{}}
}
func (f *fakeOAuthStore) CreateSession(_ context.Context, state string, s outbound.OAuthSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sessions[state]; ok {
		return errors.New("duplicate")
	}
	f.sessions[state] = s
	return nil
}
func (f *fakeOAuthStore) ConsumeSession(_ context.Context, state string) (outbound.OAuthSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.sessions[state]
	if !ok {
		return v, errors.New("missing")
	}
	delete(f.sessions, state)
	return v, nil
}
func (f *fakeOAuthStore) SaveConnection(_ context.Context, c outbound.OAuthConnection) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.links[c.OwnerID+"|"+c.GoogleSubject] = c
	return nil
}
func (f *fakeOAuthStore) GetConnection(_ context.Context, owner, subject string) (outbound.OAuthConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.links[owner+"|"+subject]
	if !ok || !c.Active {
		return outbound.OAuthConnection{}, errors.New("not found")
	}
	return c, nil
}
func (f *fakeOAuthStore) UpdateCredentials(_ context.Context, c outbound.OAuthConnection, expected int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := c.OwnerID + "|" + c.GoogleSubject
	old := f.links[k]
	if !old.Active || old.Version != expected {
		return errors.New("version conflict")
	}
	c.Version = expected + 1
	f.links[k] = c
	return nil
}
func (f *fakeOAuthStore) DeactivateConnection(_ context.Context, owner, subject string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := owner + "|" + subject
	c, ok := f.links[k]
	if !ok {
		return errors.New("not found")
	}
	c.Active = false
	c.Credentials = outbound.OAuthCredentials{}
	c.Version++
	f.links[k] = c
	return nil
}

func TestLinkServiceConsumeExactlyOnceAndScopeBinding(t *testing.T) {
	f := &fakeGoogleProvider{}
	db := newFakeOAuthStore()
	s, err := NewLinkService(f, db, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Begin(context.Background(), "", []string{"openid"}); err == nil {
		t.Fatal("untrusted empty owner accepted")
	}
	if _, err := s.Begin(context.Background(), "line-verified", nil); err == nil {
		t.Fatal("empty scope accepted")
	}
	url, err := s.Begin(context.Background(), "line-verified", []string{"https://www.googleapis.com/auth/forms.body"})
	if err != nil || url == "" {
		t.Fatal(err)
	}
	a, err := s.Complete(context.Background(), f.state, "code")
	if err != nil {
		t.Fatal(err)
	}
	if a.OwnerID != "line-verified" || a.GoogleSubject != "google-sub" {
		t.Fatal("identity mismatch")
	}
	c, err := db.GetConnection(context.Background(), "line-verified", "google-sub")
	if err != nil || len(c.Scopes) != 1 {
		t.Fatal("scope or identity lost")
	}
	if _, err := db.GetConnection(context.Background(), "other-line", "google-sub"); err == nil {
		t.Fatal("cross user access")
	}
	if _, err := s.Complete(context.Background(), f.state, "code"); err == nil {
		t.Fatal("state replay accepted")
	}
	if f.exchanges != 1 {
		t.Fatal("OAuth code exchanged more than once")
	}
}
func TestLinkServiceDisconnectFailClosed(t *testing.T) {
	p := &fakeGoogleProvider{revokeErr: errors.New("network unavailable")}
	db := newFakeOAuthStore()
	s, _ := NewLinkService(p, db, db)
	_, err := s.Begin(context.Background(), "line", []string{"openid"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Complete(context.Background(), p.state, "code"); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(context.Background(), "line", "google-sub"); err == nil {
		t.Fatal("missing warning on remote revoke failure")
	}
	if _, err := db.GetConnection(context.Background(), "line", "google-sub"); err == nil {
		t.Fatal("link was not disabled")
	}
}
