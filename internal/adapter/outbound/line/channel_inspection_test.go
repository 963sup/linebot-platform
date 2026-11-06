package line

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
)

func TestDecodeJSONValuePreservesLargeInteger(t *testing.T) {
	value, err := decodeJSONValue([]byte(`{"audienceGroupId":9007199254740993}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"audienceGroupId":9007199254740993}` {
		t.Fatalf("large integer lost precision: %s", encoded)
	}
}

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
	day := insightsreporting.DailyQuery{Day: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
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
		{"reply stats", func() error { _, e := sender.DeliveryReply(context.Background(), day); return e }},
		{"push stats", func() error { _, e := sender.DeliveryPush(context.Background(), day); return e }},
		{"multicast stats", func() error { _, e := sender.DeliveryMulticast(context.Background(), day); return e }},
		{"broadcast stats", func() error { _, e := sender.DeliveryBroadcast(context.Background(), day); return e }},
		{"narrowcast progress", func() error {
			_, e := sender.NarrowcastProgress(context.Background(), "123e4567-e89b-42d3-a456-426614174000")
			return e
		}},
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

func TestDeliveryCountResultRejectsNonObjectPayload(t *testing.T) {
	if _, err := deliveryCountResult("test", nil, "not-an-object", nil, time.Now()); err == nil {
		t.Fatal("non-object delivery count accepted")
	}
}
