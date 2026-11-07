package promotion

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrContextRequired = errors.New("context is required")

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}

const (
	DefaultCouponPageSize = 20
	MaxCouponPageSize     = 1000
)

type CouponID string

type CouponPageRequest struct {
	Cursor string
	Limit  int
}

type CouponView struct {
	ID          CouponID
	Title       string
	Description string
	Status      string
	StartsAt    time.Time
	EndsAt      time.Time
}

type CouponPage struct {
	Coupons    []CouponView
	NextCursor string
}

type CouponReader interface {
	ListCoupons(context.Context, CouponPageRequest) (CouponPage, error)
	GetCoupon(context.Context, CouponID) (CouponView, error)
}

type CouponQueries struct{ reader CouponReader }

func NewCouponQueries(reader CouponReader) (*CouponQueries, error) {
	if reader == nil {
		return nil, errors.New("coupon reader is required")
	}
	return &CouponQueries{reader: reader}, nil
}

func (q *CouponQueries) List(ctx context.Context, page CouponPageRequest) (CouponPage, error) {
	if err := validateContext(ctx); err != nil {
		return CouponPage{}, err
	}
	if page.Cursor != strings.TrimSpace(page.Cursor) || page.Limit < 0 || page.Limit > MaxCouponPageSize {
		return CouponPage{}, errors.New("invalid coupon pagination")
	}
	if page.Limit == 0 {
		page.Limit = DefaultCouponPageSize
	}
	return q.reader.ListCoupons(ctx, page)
}

func (q *CouponQueries) Get(ctx context.Context, id CouponID) (CouponView, error) {
	if err := validateContext(ctx); err != nil {
		return CouponView{}, err
	}
	if !validCouponID(id) {
		return CouponView{}, errors.New("coupon ID is required")
	}
	return q.reader.GetCoupon(ctx, id)
}

// ProviderCouponDocument preserves the complete provider coupon command.
// The outbound adapter owns SDK decoding.
type ProviderCouponDocument struct{ Value any }

type ProviderCouponCreator interface {
	CreateCouponDocument(context.Context, ProviderCouponDocument) (CouponView, error)
}

type ProviderCouponGateway interface {
	ProviderCouponCreator
	CloseCoupon(context.Context, CouponID) error
}

func validCouponID(id CouponID) bool    { return validCouponText(string(id)) }
func validCouponText(value string) bool { return value != "" && value == strings.TrimSpace(value) }
