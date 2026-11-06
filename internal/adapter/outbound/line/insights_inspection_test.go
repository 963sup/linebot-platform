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

// Real SDK envelopes exercise the anti-corruption boundary, including counters
// nested below overview, impression, metrics and per-area collections.
func TestInsightNonemptyProviderReports(t *testing.T) {
	fixtures := map[string]string{
		"/v2/bot/insight/demographic":               `{"available":true,"genders":[{"gender":"female","percentage":52.3}],"ages":[{"age":"from20to24","percentage":12.5}],"areas":[{"area":"Tokyo","percentage":9.7}],"appTypes":[{"appType":"ios","percentage":60.2}],"subscriptionPeriods":[{"subscriptionPeriod":"within7days","percentage":3.4}]}`,
		"/v2/bot/insight/followers":                 `{"status":"ready","followers":101,"targetedReaches":80,"blocks":9}`,
		"/v2/bot/insight/message/delivery":          `{"status":"ready","broadcast":1,"targeting":2,"autoResponse":3,"welcomeResponse":4,"chat":5,"apiBroadcast":6,"apiPush":7,"apiMulticast":8,"apiNarrowcast":9,"apiReply":10}`,
		"/v2/bot/insight/message/event":             `{"overview":{"requestId":"req","timestamp":1700000000,"delivered":50,"uniqueImpression":30,"uniqueClick":20,"uniqueMediaPlayed":10,"uniqueMediaPlayed100Percent":5},"messages":[{"seq":2,"impression":40,"mediaPlayed25Percent":11,"uniqueMediaPlayed100Percent":4}],"clicks":[{"seq":3,"url":"https://example.com/a","click":12,"uniqueClick":8,"uniqueClickOfRequest":7}]}`,
		"/v2/bot/insight/message/event/aggregation": `{"overview":{"uniqueImpression":32,"uniqueClick":21,"uniqueMediaPlayed":11,"uniqueMediaPlayed100Percent":6},"messages":[{"seq":4,"impression":42,"uniqueImpression":31,"mediaPlayed50Percent":13}],"clicks":[{"seq":3000000000,"url":"https://example.com/b","click":14,"uniqueClick":9,"uniqueClickOfRequest":8}]}`,
		"/v2/bot/insight/richmenu/menu/summary":     `{"richMenuId":"menu","impression":{"metrics":{"count":100,"uniqueUsers":70}},"clicks":[{"bounds":{"x":0,"y":10,"width":100,"height":50},"metrics":{"count":20,"uniqueUsers":15}},{"bounds":{"x":100,"y":10,"width":100,"height":50},"metrics":{"count":30,"uniqueUsers":25}}]}`,
		"/v2/bot/insight/richmenu/menu/daily":       `{"richMenuId":"menu","impression":{"metrics":[{"date":"20261101","count":60,"uniqueUsers":40},{"date":"20261102","count":40,"uniqueUsers":30}]},"clicks":[{"bounds":{"x":0,"y":10,"width":100,"height":50},"metrics":[{"date":"20261101","count":20,"uniqueUsers":15}]}]}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected provider request: %s", r.Method)
		}
		payload, ok := fixtures[r.URL.Path]
		if !ok {
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()
	client, err := insight.NewInsightAPI("token", insight.WithEndpoint(server.URL), insight.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewInsightReader(client)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	day := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	period := insightsreporting.RangeQuery{From: day, To: day.Add(24 * time.Hour)}
	demographics, err := reader.FriendsDemographics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !demographics.Available || len(demographics.Dimensions) != 5 {
		t.Fatalf("demographics=%+v", demographics)
	}
	for i, want := range []struct {
		dimension, value string
		ratio            float64
	}{{"gender", "female", 52.3}, {"age", "from20to24", 12.5}, {"area", "Tokyo", 9.7}, {"appType", "ios", 60.2}, {"subscriptionPeriod", "within7days", 3.4}} {
		got := demographics.Dimensions[i]
		if got.Dimension != want.dimension || got.Value != want.value || got.Ratio != want.ratio || got.Count != 0 {
			t.Fatalf("dimension=%+v want=%+v", got, want)
		}
	}
	followers, err := reader.FollowerMetrics(ctx, insightsreporting.DailyQuery{Day: day})
	if err != nil {
		t.Fatal(err)
	}
	if followers.Followers != 101 || followers.TargetedReaches != 80 || followers.Blocks != 9 || followers.CalculationStatus != "ready" || followers.Accepted != 80 || followers.Delivered != 101 || followers.Failed != 9 {
		t.Fatalf("followers/legacy=%+v", followers)
	}
	counts, err := reader.MessageDeliveries(ctx, insightsreporting.DailyQuery{Day: day})
	if err != nil {
		t.Fatal(err)
	}
	if counts.Delivered != 55 || counts.CalculationStatus != "ready" || !counts.ObservedAt.Equal(day) {
		t.Fatalf("counts=%+v", counts)
	}
	event, err := reader.MessageEvent(ctx, insightsreporting.MessageEventQuery{RequestID: "req"})
	if err != nil {
		t.Fatal(err)
	}
	if event.RequestID != "req" || event.Accepted != 0 || event.Delivered != 0 || event.Status != "" {
		t.Fatalf("legacy event projection=%+v", event)
	}
	assertInsightPoint(t, event.Points, "overview.delivered", 50, "", nil)
	index := int64(2)
	assertInsightPoint(t, event.Points, "message.mediaPlayed25Percent", 11, "", &index)
	index = 3
	assertInsightPoint(t, event.Points, "click.uniqueClickOfRequest", 7, "https://example.com/a", &index)
	if event.Points[0].At.Unix() != 1700000000 {
		t.Fatal("event delivery timestamp lost")
	}
	unit, err := reader.StatisticsPerUnit(ctx, insightsreporting.UnitStatisticsQuery{Unit: "unit", RangeQuery: period})
	if err != nil {
		t.Fatal(err)
	}
	assertInsightPoint(t, unit.Points, "overview.uniqueImpression", 32, "", nil)
	index = 4
	assertInsightPoint(t, unit.Points, "message.uniqueImpression", 31, "", &index)
	index = 3000000000
	assertInsightPoint(t, unit.Points, "click.click", 14, "https://example.com/b", &index)
	summary, err := reader.PresentationSummary(ctx, insightsreporting.PresentationStatisticsQuery{PresentationID: "menu", RangeQuery: period})
	if err != nil {
		t.Fatal(err)
	}
	assertInsightPoint(t, summary.Points, "impression.count", 100, "", nil)
	index = 0
	assertInsightPoint(t, summary.Points, "click.uniqueUsers", 15, "0,10,100,50", &index)
	index = 1
	assertInsightPoint(t, summary.Points, "click.count", 30, "100,10,100,50", &index)
	daily, err := reader.PresentationDaily(ctx, insightsreporting.PresentationStatisticsQuery{PresentationID: "menu", RangeQuery: period})
	if err != nil {
		t.Fatal(err)
	}
	if len(daily.Points) != 6 || daily.Points[0].At.Format("20060102") != "20261101" || daily.Points[2].At.Format("20060102") != "20261102" {
		t.Fatalf("daily dates=%+v", daily)
	}
	_, offset := daily.Points[0].At.Zone()
	if offset != 9*60*60 {
		t.Fatal("provider JST date lost")
	}
	index = 0
	assertInsightPoint(t, daily.Points, "click.count", 20, "0,10,100,50", &index)
}

func assertInsightPoint(t *testing.T, points []insightsreporting.MetricPoint, name string, value float64, area string, index *int64) {
	t.Helper()
	for _, p := range points {
		sameIndex := p.Index == nil && index == nil || p.Index != nil && index != nil && *p.Index == *index
		if p.Name == name && p.Area == area && sameIndex {
			if p.Value != value || p.Unit != "count" {
				t.Fatalf("point=%+v want %v", p, value)
			}
			return
		}
	}
	t.Fatalf("missing metric %s area=%s", name, area)
}

func TestInsightUnavailableOptionalAndFailureResponses(t *testing.T) {
	unavailable, err := demographicsReport(&insight.GetFriendsDemographicsResponse{Available: false})
	if err != nil || unavailable.Available || len(unavailable.Dimensions) != 0 {
		t.Fatalf("unavailable=%+v,%v", unavailable, err)
	}
	for _, v := range []any{&insight.GetStatisticsPerUnitResponse{}, &insight.GetRichMenuInsightSummaryResponse{Clicks: []insight.GetRichMenuInsightSummaryResponseClick{{}}}, &insight.GetRichMenuInsightDailyResponse{Clicks: []insight.GetRichMenuInsightDailyResponseClick{{}}}} {
		result, err := statisticsResult("test", nil, v, nil, time.Now())
		if err != nil || len(result.Points) != 0 {
			t.Fatalf("nil optionals=%+v,%v", result, err)
		}
	}
	if len(messageEventPoints(&insight.GetMessageEventResponse{}, time.Now())) != 0 {
		t.Fatal("invented event metrics")
	}
	invalid := &insight.GetRichMenuInsightDailyResponse{Impression: &insight.GetRichMenuInsightDailyResponseImpression{Metrics: []insight.GetRichMenuInsightDailyResponseDailyMetrics{{Date: "invalid"}}}}
	if _, err := statisticsResult("daily", nil, invalid, nil, time.Now()); err == nil {
		t.Fatal("invalid date accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "private-provider-body", http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := insight.NewInsightAPI("token", insight.WithEndpoint(server.URL), insight.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewInsightReader(client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.FriendsDemographics(context.Background()); err == nil {
		t.Fatal("provider failure accepted")
	}
}
