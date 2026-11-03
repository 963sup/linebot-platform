package line

import (
	"context"
	"encoding/json"
	"errors"

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

func (s *InsightReader) FriendsDemographics(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	response, result, err := client.GetFriendsDemographicsWithHttpInfo()
	return reportResult("getFriendsDemographics", response, result, err)
}
func (s *InsightReader) MessageDeliveries(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	response, result, err := client.GetNumberOfMessageDeliveriesWithHttpInfo(date)
	return reportResult("getNumberOfMessageDeliveries", response, result, err)
}
func (s *InsightReader) FollowerMetrics(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	response, result, err := client.GetNumberOfFollowersWithHttpInfo(date)
	return reportResult("getNumberOfFollowers", response, result, err)
}
func (s *InsightReader) MessageEvent(ctx context.Context, requestID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	response, result, err := client.GetMessageEventWithHttpInfo(requestID)
	return reportResult("getMessageEvent", response, result, err)
}
func (s *InsightReader) StatisticsPerUnit(ctx context.Context, unit, from, to string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	response, result, err := client.GetStatisticsPerUnitWithHttpInfo(unit, from, to)
	return reportResult("getStatisticsPerUnit", response, result, err)
}
func (s *InsightReader) RichMenuSummary(ctx context.Context, menuID, from, to string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	response, result, err := client.GetRichMenuInsightSummaryWithHttpInfo(menuID, from, to)
	return reportResult("getRichMenuInsightSummary", response, result, err)
}
func (s *InsightReader) RichMenuDaily(ctx context.Context, menuID, from, to string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	response, result, err := client.GetRichMenuInsightDailyWithHttpInfo(menuID, from, to)
	return reportResult("getRichMenuInsightDaily", response, result, err)
}
