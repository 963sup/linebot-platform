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

func TestChannelInspectorUsesRealSDKEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Errorf("unexpected LINE report request %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	sender := contractSender(t, server)
	checks := []struct {
		name string
		read func() error
	}{
		{"bot", func() error { _, e := sender.BotInfo(context.Background()); return e }},
		{"quota", func() error { _, e := sender.MessageQuota(context.Background()); return e }},
		{"consumption", func() error { _, e := sender.MessageQuotaConsumption(context.Background()); return e }},
		{"webhook", func() error { _, e := sender.WebhookEndpoint(context.Background()); return e }},
		{"aggregation", func() error { _, e := sender.AggregationUnitUsage(context.Background()); return e }},
		{"aggregation names", func() error { _, e := sender.AggregationUnitNames(context.Background(), "10", ""); return e }},
		{"reply stats", func() error { _, e := sender.DeliveryReply(context.Background(), "20261101"); return e }},
		{"push stats", func() error { _, e := sender.DeliveryPush(context.Background(), "20261101"); return e }},
		{"multicast stats", func() error { _, e := sender.DeliveryMulticast(context.Background(), "20261101"); return e }},
		{"broadcast stats", func() error { _, e := sender.DeliveryBroadcast(context.Background(), "20261101"); return e }},
		{"narrowcast progress", func() error { _, e := sender.NarrowcastProgress(context.Background(), "123e4567-e89b-42d3-a456-426614174000"); return e }},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.read(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestChannelReportErrorIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "private-token-and-message", http.StatusTooManyRequests)
	}))
	defer server.Close()
	_, err := contractSender(t, server).MessageQuota(context.Background())
	var external *resilience.ExternalError
	if !errors.As(err, &external) || external.StatusCode != 429 {
		t.Fatalf("unexpected report error: %v", err)
	}
	if strings.Contains(err.Error(), "private-token-and-message") {
		t.Fatal("SDK errors may not expose private report data")
	}
}
