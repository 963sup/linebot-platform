package insightsreporting

import (
	"context"
	"errors"
	"strings"
	"time"
)

type ReportingPeriod struct {
	From time.Time
	To   time.Time
}

type ProviderAnalyticsQuery struct {
	Metric string
	Unit   string
	Period ReportingPeriod
}

type MetricValue struct {
	Name  string
	Unit  string
	Value float64
}

type ProviderAnalytics struct {
	Source     string
	Period     ReportingPeriod
	Metrics    []MetricValue
	ObservedAt time.Time
}

type DeliveryFact struct {
	ID          string
	DeliveryID  string
	Outcome     string
	OccurredAt  time.Time
	RecipientID string
}

// ProviderAnalyticsReader maps provider-specific analytics into this
// consumer-owned representation.
type ProviderAnalyticsReader interface {
	QueryProviderAnalytics(context.Context, ProviderAnalyticsQuery) (ProviderAnalytics, error)
}

// DeliveryFactProjector must treat DeliveryFact.ID as its idempotency key.
type DeliveryFactProjector interface {
	ProjectDeliveryFact(context.Context, DeliveryFact) error
}

type Service struct {
	analytics ProviderAnalyticsReader
	projector DeliveryFactProjector
}

func NewService(analytics ProviderAnalyticsReader, projector DeliveryFactProjector) (*Service, error) {
	if analytics == nil || projector == nil {
		return nil, errors.New("analytics reader and delivery fact projector are required")
	}
	return &Service{analytics: analytics, projector: projector}, nil
}

func (s *Service) QueryProviderAnalytics(ctx context.Context, query ProviderAnalyticsQuery) (ProviderAnalytics, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderAnalytics{}, err
	}
	if !validText(query.Metric) || !validText(query.Unit) {
		return ProviderAnalytics{}, errors.New("analytics metric and unit are required")
	}
	if err := validatePeriod(query.Period); err != nil {
		return ProviderAnalytics{}, err
	}
	return s.analytics.QueryProviderAnalytics(ctx, query)
}

func (s *Service) ProjectDeliveryFact(ctx context.Context, fact DeliveryFact) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if !validText(fact.ID) || !validText(fact.DeliveryID) || !validText(fact.Outcome) || fact.OccurredAt.IsZero() {
		return errors.New("delivery fact ID, delivery ID, outcome, and occurrence time are required")
	}
	if fact.RecipientID != "" && !validText(fact.RecipientID) {
		return errors.New("invalid delivery fact recipient ID")
	}
	return s.projector.ProjectDeliveryFact(ctx, fact)
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func validatePeriod(period ReportingPeriod) error {
	if period.From.IsZero() || period.To.IsZero() || period.To.Before(period.From) {
		return errors.New("reporting period must be present and ordered")
	}
	return nil
}

func validText(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}
