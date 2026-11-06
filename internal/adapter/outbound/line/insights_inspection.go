package line

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	"github.com/line/line-bot-sdk-go/v8/linebot/insight"
)

// InsightReader copies the SDK client before setting per-call context.
type InsightReader struct {
	client *insight.InsightAPI
}

func NewInsightReader(client *insight.InsightAPI) (*InsightReader, error) {
	if client == nil {
		return nil, errors.New("LINE Insight client is required")
	}
	return &InsightReader{client: client}, nil
}

func (s *InsightReader) clientFor(ctx context.Context) (*insight.InsightAPI, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	call := *s.client
	return call.WithContext(ctx), nil
}

func (s *InsightReader) FriendsDemographics(ctx context.Context) (insightsreporting.DemographicsReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.DemographicsReport{}, err
	}
	response, result, err := client.GetFriendsDemographicsWithHttpInfo()
	if err != nil {
		return insightsreporting.DemographicsReport{}, lineReadError("getFriendsDemographics", response, err)
	}
	if result == nil {
		return insightsreporting.DemographicsReport{}, errors.New("empty LINE demographics report")
	}
	return demographicsReport(result)
}
func (s *InsightReader) MessageDeliveries(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	response, result, err := client.GetNumberOfMessageDeliveriesWithHttpInfo(query.Day.Format("20060102"))
	if err != nil {
		return insightsreporting.CountReport{}, lineReadError("getNumberOfMessageDeliveries", response, err)
	}
	if result == nil {
		return insightsreporting.CountReport{}, errors.New("empty LINE delivery report")
	}
	return countReport(result, query.Day)
}
func (s *InsightReader) FollowerMetrics(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	response, result, err := client.GetNumberOfFollowersWithHttpInfo(query.Day.Format("20060102"))
	if err != nil {
		return insightsreporting.CountReport{}, lineReadError("getNumberOfFollowers", response, err)
	}
	if result == nil {
		return insightsreporting.CountReport{}, errors.New("empty LINE follower report")
	}
	object, err := providerObject(result)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	return insightsreporting.CountReport{Accepted: int64Value(object, "targetedReaches"), Delivered: int64Value(object, "followers"), Failed: int64Value(object, "blocks"), ObservedAt: query.Day}, nil
}
func (s *InsightReader) MessageEvent(ctx context.Context, query insightsreporting.MessageEventQuery) (insightsreporting.MessageEventReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.MessageEventReport{}, err
	}
	response, result, err := client.GetMessageEventWithHttpInfo(query.RequestID)
	if err != nil {
		return insightsreporting.MessageEventReport{}, lineReadError("getMessageEvent", response, err)
	}
	if result == nil {
		return insightsreporting.MessageEventReport{}, errors.New("empty LINE message event report")
	}
	object, err := providerObject(result)
	if err != nil {
		return insightsreporting.MessageEventReport{}, err
	}
	return insightsreporting.MessageEventReport{RequestID: query.RequestID, Status: stringValue(object, "status"), Accepted: int64Value(object, "accepted"), Delivered: int64Value(object, "delivered"), ObservedAt: time.Now().UTC()}, nil
}
func (s *InsightReader) StatisticsPerUnit(ctx context.Context, query insightsreporting.UnitStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.StatisticsReport{}, err
	}
	response, result, err := client.GetStatisticsPerUnitWithHttpInfo(query.Unit, query.From.Format("20060102"), query.To.Format("20060102"))
	return statisticsResult("getStatisticsPerUnit", response, result, err, query.To)
}
func (s *InsightReader) PresentationSummary(ctx context.Context, query insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.StatisticsReport{}, err
	}
	response, result, err := client.GetRichMenuInsightSummaryWithHttpInfo(query.PresentationID, query.From.Format("20060102"), query.To.Format("20060102"))
	return statisticsResult("getRichMenuInsightSummary", response, result, err, query.To)
}
func (s *InsightReader) PresentationDaily(ctx context.Context, query insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.StatisticsReport{}, err
	}
	response, result, err := client.GetRichMenuInsightDailyWithHttpInfo(query.PresentationID, query.From.Format("20060102"), query.To.Format("20060102"))
	return statisticsResult("getRichMenuInsightDaily", response, result, err, query.To)
}

func demographicsReport(value any) (insightsreporting.DemographicsReport, error) {
	object, err := providerObject(value)
	if err != nil {
		return insightsreporting.DemographicsReport{}, err
	}
	return insightsreporting.DemographicsReport{ObservedAt: time.Now().UTC(), Dimensions: numericDimensions(object)}, nil
}

func countReport(value any, at time.Time) (insightsreporting.CountReport, error) {
	object, err := providerObject(value)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	var delivered int64
	for _, key := range []string{"broadcast", "targeting", "autoResponse", "welcomeResponse", "chat", "apiBroadcast", "apiPush", "apiMulticast", "apiNarrowcast", "apiReply"} {
		delivered += int64Value(object, key)
	}
	return insightsreporting.CountReport{Delivered: delivered, ObservedAt: at}, nil
}

func statisticsResult(op string, response *http.Response, value any, err error, at time.Time) (insightsreporting.StatisticsReport, error) {
	if err != nil {
		return insightsreporting.StatisticsReport{}, lineReadError(op, response, err)
	}
	if value == nil {
		return insightsreporting.StatisticsReport{}, errors.New("empty LINE statistics report")
	}
	object, mapErr := providerObject(value)
	if mapErr != nil {
		return insightsreporting.StatisticsReport{}, mapErr
	}
	points := make([]insightsreporting.MetricPoint, 0)
	for _, dimension := range numericDimensions(object) {
		points = append(points, insightsreporting.MetricPoint{At: at, Name: dimension.Dimension, Value: float64(dimension.Count), Unit: "count"})
	}
	return insightsreporting.StatisticsReport{Points: points, ObservedAt: at}, nil
}

func numericDimensions(object map[string]any) []insightsreporting.DimensionValue {
	keys := make([]string, 0, len(object))
	for key, raw := range object {
		switch raw.(type) {
		case json.Number, float64:
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]insightsreporting.DimensionValue, 0, len(keys))
	for _, key := range keys {
		result = append(result, insightsreporting.DimensionValue{Dimension: key, Count: int64Value(object, key)})
	}
	return result
}

var _ insightsreporting.ReportReader = (*InsightReader)(nil)
