package engagement

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// MembershipQuery contains only membership operations; coupons have their own port.
type MembershipQuery struct {
	Kind         string
	UserID       string
	MembershipID int
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
		if q.MembershipID <= 0 || q.MembershipID > 2147483647 {
			return errors.New("membership ID must be in range 1-2147483647")
		}
		return nil
	default:
		return errors.New("unsupported membership query kind")
	}
}

func (s *MembershipInspector) Inspect(ctx context.Context, q MembershipQuery) (outbound.LINEResult, error) {
	if ctx == nil {
		return outbound.LINEResult{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.LINEResult{}, err
	}
	if err := ValidateMembershipQuery(q); err != nil {
		return outbound.LINEResult{}, err
	}
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	switch q.Kind {
	case "memberships":
		return s.reader.MembershipList(ctx)
	case "subscription":
		return s.reader.MembershipSubscription(ctx, q.UserID)
	default:
		return s.reader.MembershipUsers(ctx, int32(q.MembershipID), q.Start, int32(limit))
	}
}
