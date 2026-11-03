package engagement

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// CouponQuery is independent of membership eligibility and permissions.
type CouponQuery struct {
	Kind     string
	CouponID string
	Start    string
	Limit    int
}

type CouponInspector struct {
	reader outbound.CouponReader
}

func NewCouponInspector(reader outbound.CouponReader) (*CouponInspector, error) {
	if reader == nil {
		return nil, errors.New("coupon reader is required")
	}
	return &CouponInspector{reader: reader}, nil
}

func ValidateCouponQuery(q CouponQuery) error {
	if q.Limit < 0 || q.Limit > 1000 || q.Start != strings.TrimSpace(q.Start) {
		return errors.New("invalid coupon pagination")
	}
	switch q.Kind {
	case "coupons":
		if q.CouponID != "" {
			return errors.New("coupon ID is not valid for coupon list")
		}
		return nil
	case "coupon":
		if q.CouponID == "" || q.CouponID != strings.TrimSpace(q.CouponID) {
			return errors.New("LINE coupon ID is required")
		}
		return nil
	default:
		return errors.New("unsupported coupon query kind")
	}
}

func (s *CouponInspector) Inspect(ctx context.Context, q CouponQuery) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateCouponQuery(q); err != nil {
		return nil, err
	}
	if q.Kind == "coupon" {
		return s.reader.CouponDetail(ctx, q.CouponID)
	}
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	return s.reader.CouponList(ctx, q.Start, int32(limit))
}
