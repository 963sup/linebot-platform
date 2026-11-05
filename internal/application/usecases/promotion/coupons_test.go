package promotion

import (
	"context"
	"testing"
	"time"
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

type fakeCouponManager struct{ definition CouponDefinition }

func (f *fakeCouponManager) CreateCoupon(_ context.Context, definition CouponDefinition) (CouponView, error) {
	f.definition = definition
	return CouponView{ID: "coupon-1"}, nil
}
func (*fakeCouponManager) CloseCoupon(context.Context, CouponID) error { return nil }

func TestCouponOperationsValidateDefinition(t *testing.T) {
	manager := &fakeCouponManager{}
	operations, err := NewCouponOperations(manager)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	definition := CouponDefinition{Title: "Offer", StartsAt: start, EndsAt: start.Add(time.Hour)}
	if _, err := operations.Create(context.Background(), definition); err != nil || manager.definition.Title != "Offer" {
		t.Fatalf("Create() definition = %+v, error = %v", manager.definition, err)
	}
	definition.EndsAt = start
	if _, err := operations.Create(context.Background(), definition); err == nil {
		t.Fatal("Create accepted invalid period")
	}
}
