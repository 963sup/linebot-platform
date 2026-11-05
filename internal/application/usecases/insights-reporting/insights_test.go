package insightsreporting

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeAnalytics struct {
	query ProviderAnalyticsQuery
	err   error
}

func (f *fakeAnalytics) QueryProviderAnalytics(_ context.Context, query ProviderAnalyticsQuery) (ProviderAnalytics, error) {
	f.query = query
	return ProviderAnalytics{Source: "provider", Period: query.Period}, f.err
}

type fakeProjector struct {
	fact DeliveryFact
	err  error
}

func (f *fakeProjector) ProjectDeliveryFact(_ context.Context, fact DeliveryFact) error {
	f.fact = fact
	return f.err
}

func TestServiceQueriesAnalyticsAndProjectsDeliveryFacts(t *testing.T) {
	analytics, projector := &fakeAnalytics{}, &fakeProjector{}
	service, err := NewService(analytics, projector)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	query := ProviderAnalyticsQuery{Metric: "deliveries", Unit: "day", Period: ReportingPeriod{From: now.Add(-time.Hour), To: now}}
	if got, err := service.QueryProviderAnalytics(context.Background(), query); err != nil || got.Source != "provider" || analytics.query != query {
		t.Fatalf("QueryProviderAnalytics() = %+v, %v", got, err)
	}
	fact := DeliveryFact{ID: "fact-1", DeliveryID: "delivery-1", Outcome: "sent", OccurredAt: now, RecipientID: "user-1"}
	if err := service.ProjectDeliveryFact(context.Background(), fact); err != nil || projector.fact != fact {
		t.Fatalf("ProjectDeliveryFact() error = %v, fact = %+v", err, projector.fact)
	}
}

func TestServiceRejectsInvalidInputAndPropagatesErrors(t *testing.T) {
	if _, err := NewService(nil, &fakeProjector{}); err == nil {
		t.Fatal("nil analytics reader accepted")
	}
	if _, err := NewService(&fakeAnalytics{}, nil); err == nil {
		t.Fatal("nil projector accepted")
	}
	portErr := errors.New("port failed")
	now := time.Now().UTC()
	service, _ := NewService(&fakeAnalytics{err: portErr}, &fakeProjector{err: portErr})
	validQuery := ProviderAnalyticsQuery{Metric: "deliveries", Unit: "day", Period: ReportingPeriod{From: now.Add(-time.Hour), To: now}}
	if _, err := service.QueryProviderAnalytics(context.Background(), validQuery); !errors.Is(err, portErr) {
		t.Fatalf("analytics error = %v", err)
	}
	validFact := DeliveryFact{ID: "fact", DeliveryID: "delivery", Outcome: "sent", OccurredAt: now}
	if err := service.ProjectDeliveryFact(context.Background(), validFact); !errors.Is(err, portErr) {
		t.Fatalf("projection error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, call := range map[string]func() error{
		"nil context": func() error { _, err := service.QueryProviderAnalytics(nil, validQuery); return err },
		"cancelled":   func() error { return service.ProjectDeliveryFact(ctx, validFact) },
		"no metric": func() error {
			_, err := service.QueryProviderAnalytics(context.Background(), ProviderAnalyticsQuery{Unit: "day", Period: validQuery.Period})
			return err
		},
		"bad period": func() error {
			_, err := service.QueryProviderAnalytics(context.Background(), ProviderAnalyticsQuery{Metric: "m", Unit: "day", Period: ReportingPeriod{From: now, To: now.Add(-time.Hour)}})
			return err
		},
		"bad fact": func() error { return service.ProjectDeliveryFact(context.Background(), DeliveryFact{}) },
		"bad recipient": func() error {
			fact := validFact
			fact.RecipientID = " user"
			return service.ProjectDeliveryFact(context.Background(), fact)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
