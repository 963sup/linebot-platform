package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRuntimeWithoutLINE(t *testing.T) {
	r, err := NewWebhookRuntime(Config{Port: "8080"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.Close()
	rec := httptest.NewRecorder()
	r.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/webhook/line", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without credentials, got %d", rec.Code)
	}
}

func TestRuntimeWithLINEAndShutdown(t *testing.T) {
	r, err := NewWebhookRuntime(Config{
		Port: "8080", Secrets: SecretConfig{ChannelSecret: "test-secret", ChannelToken: "test-token"},
		LINEAPITimeout: 5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	r.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected healthy runtime, got %d", rec.Code)
	}
	r.Close()
	r.Close()
}

func TestRuntimeInvalidTimeout(t *testing.T) {
	_, err := NewWebhookRuntime(Config{
		Secrets:        SecretConfig{ChannelSecret: "secret", ChannelToken: "token"},
		LINEAPITimeout: -time.Second,
	}, nil)
	if err == nil {
		t.Fatal("expected invalid outbound policy to fail")
	}
}
