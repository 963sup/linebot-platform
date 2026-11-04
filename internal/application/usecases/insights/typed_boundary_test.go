package insights

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type insightsReaderStub struct{ operation string }

func (s *insightsReaderStub) result(operation string) (outbound.LINEResult, error) {
	s.operation = operation
	return outbound.LINEResult{Value: outbound.LINEObject{"operation": operation}}, nil
}
func (s *insightsReaderStub) FriendsDemographics(context.Context) (outbound.LINEResult, error) {
	return s.result("demographics")
}
func (s *insightsReaderStub) MessageDeliveries(context.Context, string) (outbound.LINEResult, error) {
	return s.result("deliveries")
}
func (s *insightsReaderStub) FollowerMetrics(context.Context, string) (outbound.LINEResult, error) {
	return s.result("followers")
}
func (s *insightsReaderStub) MessageEvent(context.Context, string) (outbound.LINEResult, error) {
	return s.result("message-event")
}
func (s *insightsReaderStub) StatisticsPerUnit(context.Context, string, string, string) (outbound.LINEResult, error) {
	return s.result("statistics-per-unit")
}
func (s *insightsReaderStub) RichMenuSummary(context.Context, string, string, string) (outbound.LINEResult, error) {
	return s.result("menu-summary")
}
func (s *insightsReaderStub) RichMenuDaily(context.Context, string, string, string) (outbound.LINEResult, error) {
	return s.result("menu-daily")
}

func TestInspectorReturnsTypedResult(t *testing.T) {
	reader := &insightsReaderStub{}
	inspector, err := NewInspector(reader)
	if err != nil {
		t.Fatal(err)
	}
	result, err := inspector.Inspect(context.Background(), Query{Kind: "deliveries", Date: "20260131"})
	if err != nil {
		t.Fatal(err)
	}
	if reader.operation != "deliveries" || result.Value == nil {
		t.Fatalf("unexpected typed result: operation=%q result=%#v", reader.operation, result)
	}
}
