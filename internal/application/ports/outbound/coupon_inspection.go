package outbound

import "context"

// CouponReader reads coupon metadata for an eligible LINE channel.
type CouponReader interface {
	CouponList(ctx context.Context, start string, limit int32) (LINEResult, error)
	CouponDetail(ctx context.Context, couponID string) (LINEResult, error)
}
