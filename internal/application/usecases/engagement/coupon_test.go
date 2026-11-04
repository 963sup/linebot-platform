package engagement

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeCouponReader struct {
	lists   int
	details int
	limit   int32
}

func (f *fakeCouponReader) CouponList(_ context.Context, _ string, limit int32) (outbound.LINEResult, error) {
	f.lists++
	f.limit = limit
	return outbound.LINEResult{Value: map[string]any{}}, nil
}

func (f *fakeCouponReader) CouponDetail(context.Context, string) (outbound.LINEResult, error) {
	f.details++
	return outbound.LINEResult{Value: map[string]any{}}, nil
}

func TestCouponInspectorValidationAndRouting(t *testing.T) {
	if _, err := NewCouponInspector(nil); err == nil {
		t.Fatal("nil reader must be rejected")
	}
	reader := &fakeCouponReader{}
	inspector, err := NewCouponInspector(reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []CouponQuery{
		{Kind: "unknown"},
		{Kind: "coupon"},
		{Kind: "coupons", CouponID: "unexpected"},
		{Kind: "coupons", Limit: 1001},
		{Kind: "coupons", Start: " invalid "},
	} {
		if err := ValidateCouponQuery(q); err == nil {
			t.Fatalf("accepted invalid coupon query: %+v", q)
		}
	}
	if _, err := inspector.Inspect(context.Background(), CouponQuery{Kind: "coupons"}); err != nil {
		t.Fatal(err)
	}
	if reader.lists != 1 || reader.limit != 20 {
		t.Fatalf("coupon list calls=%d limit=%d, want 1,20", reader.lists, reader.limit)
	}
	if _, err := inspector.Inspect(context.Background(), CouponQuery{Kind: "coupon", CouponID: "coupon-1"}); err != nil {
		t.Fatal(err)
	}
	if reader.details != 1 {
		t.Fatalf("coupon detail calls=%d, want 1", reader.details)
	}
}
