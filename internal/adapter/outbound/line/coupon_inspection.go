package line

import (
	"context"
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
)

func (s *Sender) ListCoupons(ctx context.Context, page promotion.CouponPageRequest) (promotion.CouponPage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return promotion.CouponPage{}, err
	}
	statuses := []string{} // SDK v8.23.0 dereferences this pointer even for an unfiltered request.
	response, result, err := client.ListCouponWithHttpInfo(&statuses, page.Cursor, int32(page.Limit))
	if err != nil {
		return promotion.CouponPage{}, lineReadError("listCoupon", response, err)
	}
	if result == nil {
		return promotion.CouponPage{}, errors.New("empty LINE coupon list")
	}
	out := promotion.CouponPage{NextCursor: result.Next, Coupons: make([]promotion.CouponView, 0, len(result.Items))}
	for _, item := range result.Items {
		out.Coupons = append(out.Coupons, promotion.CouponView{ID: promotion.CouponID(item.CouponId), Title: item.Title})
	}
	return out, nil
}

func (s *Sender) GetCoupon(ctx context.Context, couponID promotion.CouponID) (promotion.CouponView, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return promotion.CouponView{}, err
	}
	response, result, err := client.GetCouponDetailWithHttpInfo(string(couponID))
	if err != nil {
		return promotion.CouponView{}, lineReadError("getCouponDetail", response, err)
	}
	if result == nil {
		return promotion.CouponView{}, errors.New("empty LINE coupon")
	}
	return promotion.CouponView{
		ID: promotion.CouponID(result.CouponId), Title: result.Title,
		Description: result.Description, Status: string(result.Status),
		StartsAt: time.UnixMilli(result.StartTimestamp), EndsAt: time.UnixMilli(result.EndTimestamp),
	}, nil
}

var _ promotion.CouponReader = (*Sender)(nil)
