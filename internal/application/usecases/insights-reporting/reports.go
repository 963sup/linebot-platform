package insightsreporting

import (
	"context"
	"errors"
	"strings"
	"time"
)

type DailyQuery struct{ Day time.Time }

type RangeQuery struct {
	From time.Time
	To   time.Time
}

type UnitStatisticsQuery struct {
	Unit string
	RangeQuery
}

type PresentationStatisticsQuery struct {
	PresentationID string
	RangeQuery
}

type MessageEventQuery struct{ RequestID string }

type DimensionValue struct {
	Dimension string
	Value     string
	Count     int64
	Ratio     float64
}

type DemographicsReport struct {
	Available  bool
	Dimensions []DimensionValue
	ObservedAt time.Time
}

type CountReport struct {
	Followers         int64
	TargetedReaches   int64
	Blocks            int64
	CalculationStatus string
	Accepted          int64
	Delivered         int64
	Failed            int64
	ObservedAt        time.Time
}

type MetricPoint struct {
	Area  string
	Index *int64
	At    time.Time
	Name  string
	Value float64
	Unit  string
}

type StatisticsReport struct {
	Points     []MetricPoint
	ObservedAt time.Time
}

type MessageEventReport struct {
	Points     []MetricPoint
	RequestID  string
	Status     string
	Accepted   int64
	Delivered  int64
	ObservedAt time.Time
}

// ReportReader owns the provider-facing report vocabulary used by this
// context. An adapter maps endpoint-specific JSON into these typed reports.
type ReportReader interface {
	FriendsDemographics(context.Context) (DemographicsReport, error)
	MessageDeliveries(context.Context, DailyQuery) (CountReport, error)
	FollowerMetrics(context.Context, DailyQuery) (CountReport, error)
	MessageEvent(context.Context, MessageEventQuery) (MessageEventReport, error)
	StatisticsPerUnit(context.Context, UnitStatisticsQuery) (StatisticsReport, error)
	PresentationSummary(context.Context, PresentationStatisticsQuery) (StatisticsReport, error)
	PresentationDaily(context.Context, PresentationStatisticsQuery) (StatisticsReport, error)
}

type PhoneDeliveryReader interface {
	PNPMessageStatistics(context.Context, DailyQuery) (CountReport, error)
	PNPTemplatedMessageStatistics(context.Context, DailyQuery) (CountReport, error)
}

// DeliveryReader covers Messaging API delivery counters which are exposed by
// a different LINE client from the Insight API but belong to this context.
type DeliveryReader interface {
	DeliveryReply(context.Context, DailyQuery) (CountReport, error)
	DeliveryPush(context.Context, DailyQuery) (CountReport, error)
	DeliveryMulticast(context.Context, DailyQuery) (CountReport, error)
	DeliveryBroadcast(context.Context, DailyQuery) (CountReport, error)
}

type Reports struct{ reader ReportReader }

func NewReports(reader ReportReader) (*Reports, error) {
	if reader == nil {
		return nil, errors.New("insights report reader is required")
	}
	return &Reports{reader: reader}, nil
}

func (r *Reports) Demographics(ctx context.Context) (DemographicsReport, error) {
	if err := validateContext(ctx); err != nil {
		return DemographicsReport{}, err
	}
	return r.reader.FriendsDemographics(ctx)
}

func (r *Reports) Deliveries(ctx context.Context, query DailyQuery) (CountReport, error) {
	if err := validateContext(ctx); err != nil {
		return CountReport{}, err
	}
	if err := validateDay(query.Day); err != nil {
		return CountReport{}, err
	}
	return r.reader.MessageDeliveries(ctx, query)
}

func (r *Reports) Followers(ctx context.Context, query DailyQuery) (CountReport, error) {
	if err := validateContext(ctx); err != nil {
		return CountReport{}, err
	}
	if err := validateDay(query.Day); err != nil {
		return CountReport{}, err
	}
	return r.reader.FollowerMetrics(ctx, query)
}

func (r *Reports) MessageEvent(ctx context.Context, query MessageEventQuery) (MessageEventReport, error) {
	if err := validateContext(ctx); err != nil {
		return MessageEventReport{}, err
	}
	if !validReportText(query.RequestID) {
		return MessageEventReport{}, errors.New("message event request ID is required")
	}
	return r.reader.MessageEvent(ctx, query)
}

func (r *Reports) StatisticsPerUnit(ctx context.Context, query UnitStatisticsQuery) (StatisticsReport, error) {
	if err := validateContext(ctx); err != nil {
		return StatisticsReport{}, err
	}
	if !validReportText(query.Unit) {
		return StatisticsReport{}, errors.New("aggregation unit is required")
	}
	if err := validateRange(query.RangeQuery); err != nil {
		return StatisticsReport{}, err
	}
	return r.reader.StatisticsPerUnit(ctx, query)
}

func (r *Reports) PresentationSummary(ctx context.Context, query PresentationStatisticsQuery) (StatisticsReport, error) {
	if err := validateContext(ctx); err != nil {
		return StatisticsReport{}, err
	}
	if err := validatePresentationQuery(query); err != nil {
		return StatisticsReport{}, err
	}
	return r.reader.PresentationSummary(ctx, query)
}

func (r *Reports) PresentationDaily(ctx context.Context, query PresentationStatisticsQuery) (StatisticsReport, error) {
	if err := validateContext(ctx); err != nil {
		return StatisticsReport{}, err
	}
	if err := validatePresentationQuery(query); err != nil {
		return StatisticsReport{}, err
	}
	return r.reader.PresentationDaily(ctx, query)
}

func (r *Reports) PhoneDeliveryStatistics(ctx context.Context, query DailyQuery, templated bool) (CountReport, error) {
	if err := validateContext(ctx); err != nil {
		return CountReport{}, err
	}
	if err := validateDay(query.Day); err != nil {
		return CountReport{}, err
	}
	phone, ok := r.reader.(PhoneDeliveryReader)
	if !ok {
		return CountReport{}, errors.New("phone delivery reporting is unavailable")
	}
	if templated {
		return phone.PNPTemplatedMessageStatistics(ctx, query)
	}
	return phone.PNPMessageStatistics(ctx, query)
}

func validatePresentationQuery(query PresentationStatisticsQuery) error {
	if !validReportText(query.PresentationID) {
		return errors.New("presentation ID is required")
	}
	return validateRange(query.RangeQuery)
}

func validateRange(query RangeQuery) error {
	if err := validateDay(query.From); err != nil {
		return err
	}
	if err := validateDay(query.To); err != nil {
		return err
	}
	if query.To.Before(query.From) {
		return errors.New("reporting date range must be ordered")
	}
	return nil
}

func validateDay(day time.Time) error {
	if day.IsZero() || day.Hour() != 0 || day.Minute() != 0 || day.Second() != 0 || day.Nanosecond() != 0 {
		return errors.New("reporting day must be a date at midnight")
	}
	return nil
}

func validReportText(value string) bool { return value != "" && value == strings.TrimSpace(value) }

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}
