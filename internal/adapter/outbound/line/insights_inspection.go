package line

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	return insightsreporting.CountReport{Accepted: result.TargetedReaches, Delivered: result.Followers, Failed: result.Blocks, Followers: result.Followers, TargetedReaches: result.TargetedReaches, Blocks: result.Blocks, CalculationStatus: string(result.Status), ObservedAt: query.Day}, nil
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
	at := time.Now().UTC()
	return insightsreporting.MessageEventReport{RequestID: query.RequestID, ObservedAt: at, Points: messageEventPoints(result, at)}, nil
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

func demographicsReport(v *insight.GetFriendsDemographicsResponse) (insightsreporting.DemographicsReport, error) {
	if v == nil {
		return insightsreporting.DemographicsReport{}, errors.New("empty LINE demographics report")
	}
	out := insightsreporting.DemographicsReport{Available: v.Available, ObservedAt: time.Now().UTC()}
	for _, tile := range v.Genders {
		out.Dimensions = append(out.Dimensions, insightsreporting.DimensionValue{Dimension: "gender", Value: string(tile.Gender), Ratio: tile.Percentage})
	}
	for _, tile := range v.Ages {
		out.Dimensions = append(out.Dimensions, insightsreporting.DimensionValue{Dimension: "age", Value: string(tile.Age), Ratio: tile.Percentage})
	}
	for _, tile := range v.Areas {
		out.Dimensions = append(out.Dimensions, insightsreporting.DimensionValue{Dimension: "area", Value: string(tile.Area), Ratio: tile.Percentage})
	}
	for _, tile := range v.AppTypes {
		out.Dimensions = append(out.Dimensions, insightsreporting.DimensionValue{Dimension: "appType", Value: string(tile.AppType), Ratio: tile.Percentage})
	}
	for _, tile := range v.SubscriptionPeriods {
		out.Dimensions = append(out.Dimensions, insightsreporting.DimensionValue{Dimension: "subscriptionPeriod", Value: string(tile.SubscriptionPeriod), Ratio: tile.Percentage})
	}
	return out, nil
}
func countReport(v *insight.GetNumberOfMessageDeliveriesResponse, at time.Time) (insightsreporting.CountReport, error) {
	if v == nil {
		return insightsreporting.CountReport{}, errors.New("empty LINE delivery report")
	}
	return insightsreporting.CountReport{Delivered: v.Broadcast + v.Targeting + v.AutoResponse + v.WelcomeResponse + v.Chat + v.ApiBroadcast + v.ApiPush + v.ApiMulticast + v.ApiNarrowcast + v.ApiReply, CalculationStatus: string(v.Status), ObservedAt: at}, nil
}

// metric preserves the provider counter name and the identity of its containing
// area or message bubble. SDK structs remain private to this adapter.
func metric(at time.Time, name string, value int64, area string, index *int64) insightsreporting.MetricPoint {
	return insightsreporting.MetricPoint{At: at, Name: name, Value: float64(value), Unit: "count", Area: area, Index: index}
}
func statisticsResult(op string, response *http.Response, value any, err error, at time.Time) (insightsreporting.StatisticsReport, error) {
	if err != nil {
		return insightsreporting.StatisticsReport{}, lineReadError(op, response, err)
	}
	out := insightsreporting.StatisticsReport{ObservedAt: at, Points: make([]insightsreporting.MetricPoint, 0)}
	switch v := value.(type) {
	case *insight.GetStatisticsPerUnitResponse:
		if v == nil {
			return out, errors.New("empty LINE statistics report")
		}
		out.Points = unitPoints(v, at)
	case *insight.GetRichMenuInsightSummaryResponse:
		if v == nil {
			return out, errors.New("empty LINE statistics report")
		}
		if v.Impression != nil && v.Impression.Metrics != nil {
			m := v.Impression.Metrics
			out.Points = append(out.Points, metric(at, "impression.count", m.Count, "", nil), metric(at, "impression.uniqueUsers", m.UniqueUsers, "", nil))
		}
		for i, c := range v.Clicks {
			if c.Metrics == nil {
				continue
			}
			area := ""
			if c.Bounds != nil {
				area = fmt.Sprintf("%d,%d,%d,%d", c.Bounds.X, c.Bounds.Y, c.Bounds.Width, c.Bounds.Height)
			}
			index := int64(i)
			out.Points = append(out.Points, metric(at, "click.count", c.Metrics.Count, area, &index), metric(at, "click.uniqueUsers", c.Metrics.UniqueUsers, area, &index))
		}
	case *insight.GetRichMenuInsightDailyResponse:
		if v == nil {
			return out, errors.New("empty LINE statistics report")
		}
		appendDaily := func(rows []insight.GetRichMenuInsightDailyResponseDailyMetrics, name, area string, index *int64) error {
			for _, m := range rows {
				day, e := time.ParseInLocation("20060102", m.Date, time.FixedZone("JST", 9*60*60))
				if e != nil {
					return errors.New("invalid LINE daily metric date")
				}
				out.Points = append(out.Points, metric(day, name+".count", m.Count, area, index), metric(day, name+".uniqueUsers", m.UniqueUsers, area, index))
			}
			return nil
		}
		if v.Impression != nil {
			if err := appendDaily(v.Impression.Metrics, "impression", "", nil); err != nil {
				return out, err
			}
		}
		for i, c := range v.Clicks {
			area := ""
			if c.Bounds != nil {
				area = fmt.Sprintf("%d,%d,%d,%d", c.Bounds.X, c.Bounds.Y, c.Bounds.Width, c.Bounds.Height)
			}
			index := int64(i)
			if err := appendDaily(c.Metrics, "click", area, &index); err != nil {
				return out, err
			}
		}
	default:
		return out, errors.New("invalid LINE statistics report")
	}
	return out, nil
}
func messageEventPoints(v *insight.GetMessageEventResponse, at time.Time) []insightsreporting.MetricPoint {
	points := make([]insightsreporting.MetricPoint, 0)
	if v.Overview != nil {
		o := v.Overview
		if o.Timestamp != 0 {
			at = time.Unix(o.Timestamp, 0).UTC()
		}
		points = append(points, metric(at, "overview.delivered", o.Delivered, "", nil))
		points = append(points, metric(at, "overview.uniqueImpression", o.UniqueImpression, "", nil))
		points = append(points, metric(at, "overview.uniqueClick", o.UniqueClick, "", nil))
		points = append(points, metric(at, "overview.uniqueMediaPlayed", o.UniqueMediaPlayed, "", nil))
		points = append(points, metric(at, "overview.uniqueMediaPlayed100Percent", o.UniqueMediaPlayed100Percent, "", nil))
	}
	for _, m := range v.Messages {
		index := int64(m.Seq)
		points = append(points, metric(at, "message.impression", m.Impression, "", &index))
		points = append(points, metric(at, "message.mediaPlayed", m.MediaPlayed, "", &index))
		points = append(points, metric(at, "message.mediaPlayed25Percent", m.MediaPlayed25Percent, "", &index))
		points = append(points, metric(at, "message.mediaPlayed50Percent", m.MediaPlayed50Percent, "", &index))
		points = append(points, metric(at, "message.mediaPlayed75Percent", m.MediaPlayed75Percent, "", &index))
		points = append(points, metric(at, "message.mediaPlayed100Percent", m.MediaPlayed100Percent, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed", m.UniqueMediaPlayed, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed25Percent", m.UniqueMediaPlayed25Percent, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed50Percent", m.UniqueMediaPlayed50Percent, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed75Percent", m.UniqueMediaPlayed75Percent, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed100Percent", m.UniqueMediaPlayed100Percent, "", &index))
	}
	for _, c := range v.Clicks {
		index := int64(c.Seq)
		area := c.Url
		points = append(points, metric(at, "click.click", c.Click, area, &index))
		points = append(points, metric(at, "click.uniqueClick", c.UniqueClick, area, &index))
		points = append(points, metric(at, "click.uniqueClickOfRequest", c.UniqueClickOfRequest, area, &index))
	}
	return points
}
func unitPoints(v *insight.GetStatisticsPerUnitResponse, at time.Time) []insightsreporting.MetricPoint {
	points := make([]insightsreporting.MetricPoint, 0)
	if v.Overview != nil {
		o := v.Overview
		points = append(points, metric(at, "overview.uniqueImpression", o.UniqueImpression, "", nil))
		points = append(points, metric(at, "overview.uniqueClick", o.UniqueClick, "", nil))
		points = append(points, metric(at, "overview.uniqueMediaPlayed", o.UniqueMediaPlayed, "", nil))
		points = append(points, metric(at, "overview.uniqueMediaPlayed100Percent", o.UniqueMediaPlayed100Percent, "", nil))
	}
	for _, m := range v.Messages {
		index := int64(m.Seq)
		points = append(points, metric(at, "message.impression", m.Impression, "", &index))
		points = append(points, metric(at, "message.mediaPlayed", m.MediaPlayed, "", &index))
		points = append(points, metric(at, "message.mediaPlayed25Percent", m.MediaPlayed25Percent, "", &index))
		points = append(points, metric(at, "message.mediaPlayed50Percent", m.MediaPlayed50Percent, "", &index))
		points = append(points, metric(at, "message.mediaPlayed75Percent", m.MediaPlayed75Percent, "", &index))
		points = append(points, metric(at, "message.mediaPlayed100Percent", m.MediaPlayed100Percent, "", &index))
		points = append(points, metric(at, "message.uniqueImpression", m.UniqueImpression, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed", m.UniqueMediaPlayed, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed25Percent", m.UniqueMediaPlayed25Percent, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed50Percent", m.UniqueMediaPlayed50Percent, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed75Percent", m.UniqueMediaPlayed75Percent, "", &index))
		points = append(points, metric(at, "message.uniqueMediaPlayed100Percent", m.UniqueMediaPlayed100Percent, "", &index))
	}
	for _, c := range v.Clicks {
		index := c.Seq
		area := c.Url
		points = append(points, metric(at, "click.click", c.Click, area, &index))
		points = append(points, metric(at, "click.uniqueClick", c.UniqueClick, area, &index))
		points = append(points, metric(at, "click.uniqueClickOfRequest", c.UniqueClickOfRequest, area, &index))
	}
	return points
}

var _ insightsreporting.ReportReader = (*InsightReader)(nil)
