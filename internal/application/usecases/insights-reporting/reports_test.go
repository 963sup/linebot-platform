package insightsreporting

import (
	"context"
	"testing"
	"time"
)

type fakeReportReader struct {
	daily DailyQuery
	unit  UnitStatisticsQuery
}

func (*fakeReportReader) FriendsDemographics(context.Context) (DemographicsReport, error) {
	return DemographicsReport{}, nil
}
func (f *fakeReportReader) MessageDeliveries(_ context.Context, q DailyQuery) (CountReport, error) {
	f.daily = q
	return CountReport{}, nil
}
func (*fakeReportReader) FollowerMetrics(context.Context, DailyQuery) (CountReport, error) {
	return CountReport{}, nil
}
func (*fakeReportReader) MessageEvent(context.Context, MessageEventQuery) (MessageEventReport, error) {
	return MessageEventReport{}, nil
}
func (f *fakeReportReader) StatisticsPerUnit(_ context.Context, q UnitStatisticsQuery) (StatisticsReport, error) {
	f.unit = q
	return StatisticsReport{}, nil
}
func (*fakeReportReader) PresentationSummary(context.Context, PresentationStatisticsQuery) (StatisticsReport, error) {
	return StatisticsReport{}, nil
}
func (*fakeReportReader) PresentationDaily(context.Context, PresentationStatisticsQuery) (StatisticsReport, error) {
	return StatisticsReport{}, nil
}
func (*fakeReportReader) PhoneDeliveryStatistics(context.Context, DailyQuery, bool) (CountReport, error) {
	return CountReport{}, nil
}

func TestReportsDispatchTypedQueries(t *testing.T) {
	reader := &fakeReportReader{}
	reports, err := NewReports(reader)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	if _, err := reports.Deliveries(context.Background(), DailyQuery{Day: day}); err != nil || reader.daily.Day != day {
		t.Fatalf("Deliveries() query = %+v, error = %v", reader.daily, err)
	}
	query := UnitStatisticsQuery{Unit: "campaign", RangeQuery: RangeQuery{From: day, To: day.AddDate(0, 0, 1)}}
	if _, err := reports.StatisticsPerUnit(context.Background(), query); err != nil || reader.unit.Unit != query.Unit {
		t.Fatalf("StatisticsPerUnit() query = %+v, error = %v", reader.unit, err)
	}
}

func TestReportsRejectInvalidQueries(t *testing.T) {
	reports, _ := NewReports(&fakeReportReader{})
	day := time.Date(2027, 1, 2, 12, 0, 0, 0, time.UTC)
	if _, err := reports.Deliveries(context.Background(), DailyQuery{Day: day}); err == nil {
		t.Fatal("Deliveries accepted non-midnight day")
	}
	if _, err := reports.MessageEvent(context.Background(), MessageEventQuery{RequestID: " request"}); err == nil {
		t.Fatal("MessageEvent accepted invalid request ID")
	}
}
