package line

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
)

func TestGetProfileUsesNativeLINEAPI(t *testing.T) {
	const userID = "U12345678901234567890123456789012"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/bot/profile/"+userID {
			t.Errorf("unexpected profile request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("unexpected LINE authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userId":"` + userID + `","displayName":"Example","pictureUrl":"https://example.invalid/photo"}`))
	}))
	defer server.Close()
	profile, err := contractSender(t, server).GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != userID || profile.DisplayName != "Example" || profile.PictureURL != "https://example.invalid/photo" {
		t.Fatalf("profile not mapped: %#v", profile)
	}
}

func TestGetProfileSanitizesErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "private-profile-data", http.StatusTooManyRequests)
	}))
	defer server.Close()
	err := func() error {
		_, err := contractSender(t, server).GetProfile(context.Background(), "U123")
		return err
	}()
	var external *resilience.ExternalError
	if !errors.As(err, &external) || external.Kind != resilience.KindRateLimited || external.StatusCode != 429 {
		t.Fatalf("unexpected profile error: %v", err)
	}
	if strings.Contains(err.Error(), "private-profile-data") {
		t.Fatal("provider response body must not be exposed")
	}
}

func TestGetProfileRejectsCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("canceled lookup must not reach network")
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := contractSender(t, server).GetProfile(ctx, "U123")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}
