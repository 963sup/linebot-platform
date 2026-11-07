package miniapp

import (
	"context"
	"errors"
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
	token    string
	calls    int
}

func (v *verifierStub) Verify(_ context.Context, token string) (linesession.Identity, error) {
	v.calls++
	v.token = token
	return v.identity, v.err
}

func authenticatedRequest() *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/miniapp/api/v1/session", nil)
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Authorization", "Bearer private-token")
	return request
}

func TestAuthenticatorAttachesVerifiedIdentity(t *testing.T) {
	identity := linesession.Identity{Subject: "private-subject", ValidatedAt: time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)}
	verifier := &verifierStub{identity: identity}
	nextCalls := 0
	handler := authenticator{verifier: verifier, allowedOrigin: "https://example.com"}.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalls++
		got, ok := IdentityFromContext(r.Context())
		if !ok || got != identity {
			t.Fatalf("identity = %#v, %v", got, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, authenticatedRequest())
	if response.Code != http.StatusNoContent || nextCalls != 1 || verifier.calls != 1 || verifier.token != "private-token" {
		t.Fatalf("status=%d next=%d verifier=%d token=%q", response.Code, nextCalls, verifier.calls, verifier.token)
	}
}

func TestAuthenticatorRejectsInvalidBrowserAndBearerInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*http.Request)
		status int
	}{
		{name: "missing origin metadata", mutate: func(r *http.Request) { r.Header.Del("Origin"); r.Header.Del("Sec-Fetch-Site") }, status: http.StatusForbidden},
		{name: "foreign origin", mutate: func(r *http.Request) { r.Header.Set("Origin", "https://attacker.example") }, status: http.StatusForbidden},
		{name: "cross site", mutate: func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, status: http.StatusForbidden},
		{name: "duplicate origin", mutate: func(r *http.Request) { r.Header.Add("Origin", "https://example.com") }, status: http.StatusForbidden},
		{name: "missing bearer", mutate: func(r *http.Request) { r.Header.Del("Authorization") }, status: http.StatusUnauthorized},
		{name: "duplicate bearer", mutate: func(r *http.Request) { r.Header.Add("Authorization", "Bearer second") }, status: http.StatusUnauthorized},
		{name: "wrong scheme", mutate: func(r *http.Request) { r.Header.Set("Authorization", "Basic token") }, status: http.StatusUnauthorized},
		{name: "empty bearer", mutate: func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") }, status: http.StatusUnauthorized},
		{name: "spaced bearer", mutate: func(r *http.Request) { r.Header.Set("Authorization", "Bearer two tokens") }, status: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := &verifierStub{}
			nextCalls := 0
			handler := authenticator{verifier: verifier, allowedOrigin: "https://example.com"}.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalls++ }))
			request := authenticatedRequest()
			test.mutate(request)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || verifier.calls != 0 || nextCalls != 0 {
				t.Fatalf("status=%d verifier=%d next=%d", response.Code, verifier.calls, nextCalls)
			}
			if strings.Contains(response.Body.String(), "private-token") {
				t.Fatalf("response leaked bearer token: %s", response.Body.String())
			}
		})
	}
}

func TestAuthenticatorMapsVerificationFailuresWithoutLeaks(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "unauthenticated", err: linesession.ErrUnauthenticated, status: http.StatusUnauthorized},
		{name: "cancelled", err: context.Canceled, status: http.StatusRequestTimeout},
		{name: "deadline", err: context.DeadlineExceeded, status: http.StatusGatewayTimeout},
		{name: "unavailable", err: errors.New("provider detail"), status: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := &verifierStub{err: test.err}
			nextCalls := 0
			handler := authenticator{verifier: verifier, allowedOrigin: "https://example.com"}.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalls++ }))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, authenticatedRequest())
			if response.Code != test.status || verifier.calls != 1 || nextCalls != 0 {
				t.Fatalf("status=%d verifier=%d next=%d", response.Code, verifier.calls, nextCalls)
			}
			body := response.Body.String()
			if strings.Contains(body, "private-token") || strings.Contains(body, "provider detail") {
				t.Fatalf("response leaked verification input: %s", body)
			}
		})
	}
}
