package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
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

func (s *InsightReader) FriendsDemographics(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetFriendsDemographicsWithHttpInfo()
	return typedReportResult("getFriendsDemographics", response, result, err)
}
func (s *InsightReader) MessageDeliveries(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetNumberOfMessageDeliveriesWithHttpInfo(date)
	return typedReportResult("getNumberOfMessageDeliveries", response, result, err)
}
func (s *InsightReader) FollowerMetrics(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetNumberOfFollowersWithHttpInfo(date)
	return typedReportResult("getNumberOfFollowers", response, result, err)
}
func (s *InsightReader) MessageEvent(ctx context.Context, requestID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetMessageEventWithHttpInfo(requestID)
	return typedReportResult("getMessageEvent", response, result, err)
}
func (s *InsightReader) StatisticsPerUnit(ctx context.Context, unit, from, to string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetStatisticsPerUnitWithHttpInfo(unit, from, to)
	return typedReportResult("getStatisticsPerUnit", response, result, err)
}
func (s *InsightReader) RichMenuSummary(ctx context.Context, menuID, from, to string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuInsightSummaryWithHttpInfo(menuID, from, to)
	return typedReportResult("getRichMenuInsightSummary", response, result, err)
}
func (s *InsightReader) RichMenuDaily(ctx context.Context, menuID, from, to string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuInsightDailyWithHttpInfo(menuID, from, to)
	return typedReportResult("getRichMenuInsightDaily", response, result, err)
}
