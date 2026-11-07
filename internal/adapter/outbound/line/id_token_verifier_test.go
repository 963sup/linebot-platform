package line

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	linesession "github.com/963sup/linebot-platform/internal/application/usecases/line-session"
)

func testIDTokenVerifier(t *testing.T, handler http.HandlerFunc) (*IDTokenVerifier, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	verifier, err := newIDTokenVerifier(&http.Client{Timeout: time.Second}, server.URL+"/oauth2/v2.1/verify", "channel")
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	return verifier, server.Close
}

func TestIDTokenVerifierPostsFormAndReturnsVerifiedSubject(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	verifier, closeServer := testIDTokenVerifier(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth2/v2.1/verify" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil || r.Form.Get("id_token") != "token" || r.Form.Get("client_id") != "channel" {
			t.Errorf("form = %v, err=%v", r.Form, err)
		}
		fmt.Fprintf(w, `{"iss":"https://access.line.me","aud":"channel","exp":%d,"sub":"U-secret","iat":1,"name":"allowed"}`, now.Add(time.Minute).Unix())
	})
	defer closeServer()
	verifier.now = func() time.Time { return now }
	principal, err := verifier.Verify(context.Background(), "token")
	if err != nil || principal.ValidatedAt != now || principal.Subject != "U-secret" {
		t.Fatalf("Verify() = (%+v, %v)", principal, err)
	}
}

func TestIDTokenVerifierRejectsClaimsAndClassifiesFailures(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name, body string
		status     int
		want       error
	}{
		{"issuer", `{"iss":"wrong","aud":"channel","exp":9999999999,"sub":"U"}`, 200, linesession.ErrUnauthenticated},
		{"audience", `{"iss":"https://access.line.me","aud":"wrong","exp":9999999999,"sub":"U"}`, 200, linesession.ErrUnauthenticated},
		{"expired", fmt.Sprintf(`{"iss":"https://access.line.me","aud":"channel","exp":%d,"sub":"U"}`, now.Unix()), 200, linesession.ErrUnauthenticated},
		{"subject", `{"iss":"https://access.line.me","aud":"channel","exp":9999999999,"sub":""}`, 200, linesession.ErrUnauthenticated},
		{"invalid token", `{}`, 400, linesession.ErrUnauthenticated},
		{"outage", `{}`, 503, linesession.ErrUnavailable},
		{"rate limit", `{}`, 429, linesession.ErrUnavailable},
		{"bounded", strings.Repeat("x", maxIDTokenResponseBytes+1), 200, linesession.ErrUnavailable},
		{"trailing", `{"iss":"https://access.line.me","aud":"channel","exp":9999999999,"sub":"U"}{}`, 200, linesession.ErrUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier, closeServer := testIDTokenVerifier(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			})
			defer closeServer()
			verifier.now = func() time.Time { return now }
			if _, err := verifier.Verify(context.Background(), "token"); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestIDTokenVerifierRequiresTimeoutAndPropagatesCancellation(t *testing.T) {
	if _, err := NewIDTokenVerifier(&http.Client{}, "channel"); err == nil {
		t.Fatal("client without timeout accepted")
	}
	verifier, closeServer := testIDTokenVerifier(t, func(http.ResponseWriter, *http.Request) {})
	defer closeServer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := verifier.Verify(ctx, "token"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}
