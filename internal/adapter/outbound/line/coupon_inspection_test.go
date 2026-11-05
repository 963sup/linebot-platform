package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
)

func TestCouponReadOnlySDKEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected coupon method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"coupons", func() error { _, err := s.ListCoupons(ctx, promotion.CouponPageRequest{Limit: 10}); return err }},
		{"coupon", func() error { _, err := s.GetCoupon(ctx, promotion.CouponID("coupon-1")); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
