package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/insight"
)

func TestInsightsReadOnlySDKEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected insights method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	client, err := insight.NewInsightAPI("private-token", insight.WithEndpoint(server.URL), insight.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewInsightReader(client)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"demographics", func() error { _, e := s.FriendsDemographics(ctx); return e }},
		{"deliveries", func() error { _, e := s.MessageDeliveries(ctx, "20261101"); return e }},
		{"followers", func() error { _, e := s.FollowerMetrics(ctx, "20261101"); return e }},
		{"message-event", func() error { _, e := s.MessageEvent(ctx, "request-1"); return e }},
		{"stats", func() error { _, e := s.StatisticsPerUnit(ctx, "unit", "20261101", "20261102"); return e }},
		{"summary", func() error { _, e := s.RichMenuSummary(ctx, "richmenu-1", "20261101", "20261102"); return e }},
		{"daily", func() error { _, e := s.RichMenuDaily(ctx, "richmenu-1", "20261101", "20261102"); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
