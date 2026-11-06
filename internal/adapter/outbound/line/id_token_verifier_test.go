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

	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
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

func TestIDTokenVerifierPostsFormAndReturnsProofWithoutSubject(t *testing.T) {
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
	if err != nil || principal.ValidatedAt != now || !principal.ParticipantID.Valid() || strings.Contains(string(principal.ParticipantID), "U-secret") || !strings.HasPrefix(string(principal.ParticipantID), "line:v1:") {
		t.Fatalf("Verify() = (%+v, %v)", principal, err)
	}
	if principal.ParticipantID != participantID("channel", "U-secret") || principal.ParticipantID == participantID("other-channel", "U-secret") {
		t.Fatal("participant identity is not stable and client-scoped")
	}
}

func TestIDTokenVerifierRejectsClaimsAndClassifiesFailures(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name, body string
		status     int
		want       error
	}{
		{"issuer", `{"iss":"wrong","aud":"channel","exp":9999999999,"sub":"U"}`, 200, attendance.ErrUnauthenticated},
		{"audience", `{"iss":"https://access.line.me","aud":"wrong","exp":9999999999,"sub":"U"}`, 200, attendance.ErrUnauthenticated},
		{"expired", fmt.Sprintf(`{"iss":"https://access.line.me","aud":"channel","exp":%d,"sub":"U"}`, now.Unix()), 200, attendance.ErrUnauthenticated},
		{"subject", `{"iss":"https://access.line.me","aud":"channel","exp":9999999999,"sub":""}`, 200, attendance.ErrUnauthenticated},
		{"invalid token", `{}`, 400, attendance.ErrUnauthenticated},
		{"outage", `{}`, 503, attendance.ErrIdentityUnavailable},
		{"rate limit", `{}`, 429, attendance.ErrIdentityUnavailable},
		{"bounded", strings.Repeat("x", maxIDTokenResponseBytes+1), 200, attendance.ErrIdentityUnavailable},
		{"trailing", `{"iss":"https://access.line.me","aud":"channel","exp":9999999999,"sub":"U"}{}`, 200, attendance.ErrIdentityUnavailable},
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
