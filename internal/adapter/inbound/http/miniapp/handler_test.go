package miniapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	linesession "github.com/963sup/linebot-platform/internal/application/usecases/line-session"
)

type verifierStub struct {
	identity linesession.Identity
	err      error
}

func (v verifierStub) Verify(context.Context, string) (linesession.Identity, error) {
	return v.identity, v.err
}

func testHandler(t *testing.T, verifier SessionVerifier) *Handler {
	t.Helper()
	handler, err := New(Config{LIFFID: "123456-example", AllowedOrigin: "https://example.com"}, verifier)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestSessionReturnsOnlyVerificationState(t *testing.T) {
	handler := testHandler(t, verifierStub{identity: linesession.Identity{Subject: "private-subject", ValidatedAt: time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)}})
	request := httptest.NewRequest(http.MethodGet, "/miniapp/api/v1/session", nil)
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"authenticated":true`) || strings.Contains(response.Body.String(), "private-subject") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSessionRejectsUntrustedRequests(t *testing.T) {
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Header.Set("Origin", "https://attacker.example") },
		func(r *http.Request) { r.Header.Del("Authorization") },
	} {
		request := httptest.NewRequest(http.MethodGet, "/miniapp/api/v1/session", nil)
		request.Header.Set("Origin", "https://example.com")
		request.Header.Set("Sec-Fetch-Site", "same-origin")
		request.Header.Set("Authorization", "Bearer token")
		mutate(request)
		response := httptest.NewRecorder()
		testHandler(t, verifierStub{}).ServeHTTP(response, request)
		if response.Code < 400 {
			t.Fatalf("untrusted request accepted: %d", response.Code)
		}
	}
}

func TestAssetsExposeNeutralLIFFAndGeolocationInfrastructure(t *testing.T) {
	for _, path := range []string{"/miniapp/", "/miniapp/assets/app.js"} {
		response := httptest.NewRecorder()
		testHandler(t, verifierStub{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != 200 {
			t.Fatalf("%s status=%d", path, response.Code)
		}
		body := response.Body.String()
		if !strings.Contains(body, "MINI App") && !strings.Contains(body, "getCurrentPosition") {
			t.Fatalf("neutral infrastructure marker missing from %s", path)
		}
	}
}
