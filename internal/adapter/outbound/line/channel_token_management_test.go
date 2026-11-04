package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/channel_access_token"
)

func TestChannelTokenAdapterUsesOfficialOAuthEndpoints(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	client, err := channel_access_token.NewChannelAccessTokenAPI(channel_access_token.WithEndpoint(server.URL), channel_access_token.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewChannelTokenManager(client)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cases := []struct {
		want string
		call func() error
	}{
		{"POST /oauth2/v3/token", func() error { _, e := s.IssueStateless(ctx, "client_credentials", "", "", "id", "secret"); return e }},
		{"GET /oauth2/v2.1/tokens/kid", func() error { _, e := s.ValidKeyIDs(ctx, "jwt-bearer", "assertion"); return e }},
		{"POST /oauth2/v2.1/token", func() error { _, e := s.IssueJWT(ctx, "client_credentials", "jwt-bearer", "assertion"); return e }},
		{"GET /oauth2/v2.1/verify", func() error { _, e := s.VerifyJWT(ctx, "token"); return e }},
		{"POST /oauth2/v2.1/revoke", func() error { return s.RevokeJWT(ctx, "id", "secret", "token") }},
		{"POST /v2/oauth/accessToken", func() error { _, e := s.IssueShortLived(ctx, "client_credentials", "id", "secret"); return e }},
		{"POST /v2/oauth/verify", func() error { _, e := s.VerifyShortLived(ctx, "token"); return e }},
		{"POST /v2/oauth/revoke", func() error { return s.RevokeShortLived(ctx, "token") }},
	}
	for _, tc := range cases {
		got = ""
		if err := tc.call(); err != nil {
			t.Fatalf("%s: %v", tc.want, err)
		}
		if got != tc.want {
			t.Fatalf("got %s, want %s", got, tc.want)
		}
	}
}

func TestChannelTokenAdapterReturnsTransportNeutralResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"secret","expires_in":3600}`))
	}))
	defer server.Close()
	client, err := channel_access_token.NewChannelAccessTokenAPI(channel_access_token.WithEndpoint(server.URL), channel_access_token.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewChannelTokenManager(client)
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.IssueShortLived(context.Background(), "client_credentials", "id", "secret")
	if err != nil {
		t.Fatal(err)
	}
	value, ok := result.Value.(map[string]any)
	if !ok || value["access_token"] != "secret" {
		t.Fatalf("unexpected typed result: %#v", result.Value)
	}
}
