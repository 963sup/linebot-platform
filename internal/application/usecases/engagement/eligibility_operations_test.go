package engagement

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeEligibility struct{ called string }

func (f *fakeEligibility) PushMessagesByPhone(context.Context, outbound.LINERequest) error {
	f.called = "push-phone"
	return nil
}
func (f *fakeEligibility) PushTemplatedMessagesByPhone(context.Context, outbound.LINERequest) error {
	f.called = "push-template-phone"
	return nil
}
func (f *fakeEligibility) PNPMessageStatistics(context.Context, string) (outbound.LINEResult, error) {
	f.called = "pnp-stats"
	return outbound.LINEResult{Value: map[string]any{}}, nil
}
func (f *fakeEligibility) PNPTemplatedMessageStatistics(context.Context, string) (outbound.LINEResult, error) {
	f.called = "pnp-template-stats"
	return outbound.LINEResult{Value: map[string]any{}}, nil
}
func (f *fakeEligibility) CreateCoupon(context.Context, outbound.LINERequest) (outbound.LINEResult, error) {
	f.called = "create-coupon"
	return outbound.LINEResult{Value: map[string]any{}}, nil
}
func (f *fakeEligibility) CloseCoupon(context.Context, string) error {
	f.called = "close-coupon"
	return nil
}

func TestEligibilityOperationsRejectInvalidAndDispatch(t *testing.T) {
	if _, err := NewEligibilityOperations(nil); err == nil {
		t.Fatal("nil port accepted")
	}
	f := &fakeEligibility{}
	s, err := NewEligibilityOperations(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []EligibilityCommand{
		{Action: "close-coupon"},
		{Action: "pnp-stats", ID: "20261301"},
		{Action: "pnp-stats", ID: "20261101", Payload: outbound.LINERequest{Value: map[string]any{}}},
		{Action: "push-phone", Payload: outbound.LINERequest{Value: []any{}}},
		{Action: "delete-everything"},
	} {
		if _, err := s.Execute(context.Background(), c); err == nil {
			t.Fatalf("accepted invalid %+v", c)
		}
		if f.called != "" {
			t.Fatal("invalid input caused side effect")
		}
	}
	for _, c := range []EligibilityCommand{
		{Action: "push-phone", Payload: outbound.LINERequest{Value: map[string]any{}}},
		{Action: "push-template-phone", Payload: outbound.LINERequest{Value: map[string]any{}}},
		{Action: "pnp-stats", ID: "20261101"},
		{Action: "pnp-template-stats", ID: "20261101"},
		{Action: "create-coupon", Payload: outbound.LINERequest{Value: map[string]any{}}},
		{Action: "close-coupon", ID: "coupon-1"},
	} {
		if _, err := s.Execute(context.Background(), c); err != nil {
			t.Fatal(err)
		}
		if f.called != c.Action {
			t.Fatalf("called %s for %s", f.called, c.Action)
		}
	}
}
