package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

func TestOAuthAuthorizationURLAndConfig(t *testing.T) {
	if _, err := NewFlow(Settings{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://example.com/callback"}); err == nil {
		t.Fatal("insecure redirect accepted")
	}
	f, err := NewFlow(Settings{ClientID: "client-id", ClientSecret: "private", RedirectURL: "https://example.test/callback"})
	if err != nil {
		t.Fatal(err)
	}
	state, err := RandomState()
	if err != nil || len(state) < 32 {
		t.Fatal("state generation failed")
	}
	nonce, err := RandomState()
	if err != nil {
		t.Fatal(err)
	}
	verifier := oauth2.GenerateVerifier()
	redirect, err := f.AuthorizationURL(state, verifier, nonce, []string{"email", "https://www.googleapis.com/auth/forms.body.readonly"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "accounts.google.com" || q.Get("state") != state || q.Get("nonce") != nonce ||
		q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" ||
		q.Get("access_type") != "offline" || !strings.Contains(q.Get("scope"), "forms.body.readonly") {
		t.Fatal("OAuth PKCE/offline authorization parameters lost")
	}
	if _, err := f.AuthorizationURL("", verifier, nonce, []string{"openid"}); err == nil {
		t.Fatal("missing state accepted")
	}
	if _, err := f.AuthorizationURL(state, verifier, nonce, []string{"https://untrusted.example/scope"}); err == nil {
		t.Fatal("untrusted scopes accepted")
	}
}

func TestOAuthExchangeVerifiesNonceAndIdentity(t *testing.T) {
	nonce, err := RandomState()
	if err != nil {
		t.Fatal(err)
	}
	verifier := oauth2.GenerateVerifier()
	var gotVerifier bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		gotVerifier = r.Form.Get("code_verifier") == verifier
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-access", "refresh_token": "test-refresh", "token_type": "Bearer", "expires_in": 3600, "id_token": "fake-id-token"})
	}))
	defer server.Close()
	f, err := NewFlow(Settings{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://example.test/callback"})
	if err != nil {
		t.Fatal(err)
	}
	f.config.Endpoint = oauth2.Endpoint{TokenURL: server.URL, AuthURL: server.URL, AuthStyle: oauth2.AuthStyleInParams}
	f.verify = func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		return &idtoken.Payload{
			Issuer: "https://accounts.google.com", Audience: audience, Subject: "stable-subject", Expires: time.Now().Add(time.Hour).Unix(),
			Claims: map[string]interface{}{"nonce": nonce, "email": "test@example.test", "email_verified": true},
		}, nil
	}
	result, err := f.ExchangeAndVerify(context.Background(), "auth-code", verifier, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !gotVerifier || result.User.Subject != "stable-subject" || result.Token.RefreshToken != "test-refresh" {
		t.Fatal("exchange result invalid")
	}
	_, err = f.ExchangeAndVerify(context.Background(), "auth-code", verifier, "wrong-nonce-value-aaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatal("wrong nonce accepted")
	}
}

func TestGoogleIDTokenRejectsImpersonation(t *testing.T) {
	nonce, err := RandomState()
	if err != nil {
		t.Fatal(err)
	}
	base := func(ctx context.Context, token, aud string) (*idtoken.Payload, error) {
		return &idtoken.Payload{Issuer: "https://accounts.google.com", Audience: aud, Subject: "sub", Expires: time.Now().Add(time.Hour).Unix(), Claims: map[string]interface{}{"nonce": nonce}}, nil
	}
	if _, err := verifyUser(context.Background(), "signed", "id", nonce, base); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		modify func(*idtoken.Payload)
	}{
		{"issuer", func(p *idtoken.Payload) { p.Issuer = "https://evil.invalid" }},
		{"subject", func(p *idtoken.Payload) { p.Subject = "" }},
		{"audience", func(p *idtoken.Payload) { p.Audience = "other-client" }},
		{"expired", func(p *idtoken.Payload) { p.Expires = time.Now().Add(-time.Hour).Unix() }},
		{"nonce", func(p *idtoken.Payload) { p.Claims["nonce"] = "another-nonce" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verify := func(ctx context.Context, raw, aud string) (*idtoken.Payload, error) {
				p, _ := base(ctx, raw, aud)
				tc.modify(p)
				return p, nil
			}
			if _, err := verifyUser(context.Background(), "signed", "id", nonce, verify); err == nil {
				t.Fatal("invalid ID token accepted")
			}
		})
	}
}
