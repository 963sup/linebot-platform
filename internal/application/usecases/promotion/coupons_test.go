package promotion

import (
	"context"
	"testing"
)

type fakeCouponReader struct{ page CouponPageRequest }

func (f *fakeCouponReader) ListCoupons(_ context.Context, page CouponPageRequest) (CouponPage, error) {
	f.page = page
	return CouponPage{}, nil
}
func (*fakeCouponReader) GetCoupon(_ context.Context, id CouponID) (CouponView, error) {
	return CouponView{ID: id}, nil
}

func TestCouponQueriesNormalizePagination(t *testing.T) {
	reader := &fakeCouponReader{}
	queries, err := NewCouponQueries(reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queries.List(context.Background(), CouponPageRequest{}); err != nil || reader.page.Limit != DefaultCouponPageSize {
		t.Fatalf("List() page = %+v, error = %v", reader.page, err)
	}
	if _, err := queries.Get(context.Background(), " "); err == nil {
		t.Fatal("Get accepted invalid coupon ID")
	}
}
