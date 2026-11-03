package engagement

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type MembershipQuery struct {
	Kind         string
	UserID       string
	MembershipID int
	CouponID     string
	Start        string
	Limit        int
}

type MembershipInspector struct {
	reader outbound.MembershipReader
}

func NewMembershipInspector(reader outbound.MembershipReader) (*MembershipInspector, error) {
	if reader == nil {
		return nil, errors.New("membership reader is required")
	}
	return &MembershipInspector{reader: reader}, nil
}

func ValidateMembershipQuery(q MembershipQuery) error {
	if q.Limit < 0 || q.Limit > 1000 || q.Start != strings.TrimSpace(q.Start) {
		return errors.New("invalid membership pagination")
	}
	switch q.Kind {
	case "memberships":
		return nil
	case "subscription":
		if q.UserID == "" || q.UserID != strings.TrimSpace(q.UserID) {
			return errors.New("LINE user ID is required")
		}
		return nil
	case "membership-users":
		if q.MembershipID <= 0 {
			return errors.New("membership ID must be positive")
		}
		if q.MembershipID > 2147483647 {
			return errors.New("membership ID out of range")
		}
		return nil
	case "coupons":
		return nil
	case "coupon":
		if q.CouponID == "" || q.CouponID != strings.TrimSpace(q.CouponID) {
			return errors.New("coupon ID is required")
		}
		return nil
	default:
		return errors.New("unsupported membership/coupon query kind")
	}
}

func (s *MembershipInspector) Inspect(ctx context.Context, q MembershipQuery) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateMembershipQuery(q); err != nil {
		return nil, err
	}
	switch q.Kind {
	case "memberships":
		return s.reader.MembershipList(ctx)
	case "subscription":
		return s.reader.MembershipSubscription(ctx, q.UserID)
	case "membership-users":
		return s.reader.MembershipUsers(ctx, int32(q.MembershipID), q.Start, int32(q.Limit))
	case "coupons":
		return s.reader.CouponList(ctx, q.Start, int32(q.Limit))
	default:
		return s.reader.CouponDetail(ctx, q.CouponID)
	}
}
