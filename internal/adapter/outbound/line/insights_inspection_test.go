package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
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
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"demographics", func() error { _, e := s.FriendsDemographics(ctx); return e }},
		{"deliveries", func() error { _, e := s.MessageDeliveries(ctx, insightsreporting.DailyQuery{Day: from}); return e }},
		{"followers", func() error { _, e := s.FollowerMetrics(ctx, insightsreporting.DailyQuery{Day: from}); return e }},
		{"message-event", func() error {
			_, e := s.MessageEvent(ctx, insightsreporting.MessageEventQuery{RequestID: "request-1"})
			return e
		}},
		{"stats", func() error {
			_, e := s.StatisticsPerUnit(ctx, insightsreporting.UnitStatisticsQuery{Unit: "unit", RangeQuery: insightsreporting.RangeQuery{From: from, To: to}})
			return e
		}},
		{"summary", func() error {
			_, e := s.PresentationSummary(ctx, insightsreporting.PresentationStatisticsQuery{PresentationID: "richmenu-1", RangeQuery: insightsreporting.RangeQuery{From: from, To: to}})
			return e
		}},
		{"daily", func() error {
			_, e := s.PresentationDaily(ctx, insightsreporting.PresentationStatisticsQuery{PresentationID: "richmenu-1", RangeQuery: insightsreporting.RangeQuery{From: from, To: to}})
			return e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
