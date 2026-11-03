package outbound

import (
	"context"
	"encoding/json"
)

// CouponReader reads coupon metadata for an eligible LINE channel.
type CouponReader interface {
	CouponList(ctx context.Context, start string, limit int32) (json.RawMessage, error)
	CouponDetail(ctx context.Context, couponID string) (json.RawMessage, error)
}
