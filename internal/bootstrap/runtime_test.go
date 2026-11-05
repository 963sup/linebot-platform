package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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
	ready := httptest.NewRecorder()
	r.Handler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("expected ready runtime, got %d", ready.Code)
	}
	r.Close()
	notReady := httptest.NewRecorder()
	r.Handler.ServeHTTP(notReady, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if notReady.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected closed runtime not ready, got %d", notReady.Code)
	}
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

func TestWebhookRuntimeCloseAggregatesErrorsOnceInReverseOwnershipOrder(t *testing.T) {
	firstErr := errors.New("first failed")
	secondErr := errors.New("second failed")
	var order []string
	runtime := &WebhookRuntime{closers: []runtimeCloser{
		{name: "first resource", close: func() error {
			order = append(order, "first")
			return firstErr
		}},
		{name: "second resource", close: func() error {
			order = append(order, "second")
			return secondErr
		}},
	}}
	runtime.ready.Store(true)

	err := runtime.Close()
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("expected both close failures, got %v", err)
	}
	if !strings.Contains(err.Error(), "first resource") || !strings.Contains(err.Error(), "second resource") {
		t.Fatalf("expected actionable resource names, got %v", err)
	}
	if strings.Contains(err.Error(), firstErr.Error()) || strings.Contains(err.Error(), secondErr.Error()) {
		t.Fatalf("close error leaked raw provider detail: %v", err)
	}
	if want := []string{"second", "first"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("close order = %v, want %v", order, want)
	}
	if runtime.ready.Load() {
		t.Fatal("closed runtime must fail readiness")
	}

	secondCloseErr := runtime.Close()
	if secondCloseErr != err {
		t.Fatalf("second close error = %v, want cached %v", secondCloseErr, err)
	}
	if want := []string{"second", "first"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("resources closed more than once: %v", order)
	}
}
