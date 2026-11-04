package engagement

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// EligibilityCommand is operator-only. The caller must verify entitlement to
// the specific LINE feature before invoking this use case.
type EligibilityCommand struct {
	Action  string
	ID      string
	Payload outbound.LINERequest
}

type EligibilityOperations struct {
	operations outbound.EligibilityOperations
}

func NewEligibilityOperations(o outbound.EligibilityOperations) (*EligibilityOperations, error) {
	if o == nil {
		return nil, errors.New("eligibility operations are required")
	}
	return &EligibilityOperations{operations: o}, nil
}

func ValidateEligibilityCommand(c EligibilityCommand) error {
	if c.ID != strings.TrimSpace(c.ID) {
		return errors.New("invalid identifier")
	}
	switch c.Action {
	case "push-phone", "push-template-phone", "create-coupon":
		if c.ID != "" || !isLINEObject(c.Payload) {
			return errors.New("action requires a JSON object and no ID")
		}
	case "pnp-stats", "pnp-template-stats":
		if len(c.ID) != 8 || c.Payload.Value != nil {
			return errors.New("statistics require YYYYMMDD and no payload")
		}
		if _, err := time.Parse("20060102", c.ID); err != nil {
			return errors.New("invalid statistics date")
		}
	case "close-coupon":
		if c.ID == "" || c.Payload.Value != nil {
			return errors.New("coupon ID and no payload are required")
		}
	default:
		return errors.New("unsupported eligibility action")
	}
	return nil
}

func (s *EligibilityOperations) Execute(ctx context.Context, c EligibilityCommand) (outbound.LINEResult, error) {
	if ctx == nil {
		return outbound.LINEResult{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.LINEResult{}, err
	}
	if err := ValidateEligibilityCommand(c); err != nil {
		return outbound.LINEResult{}, err
	}
	switch c.Action {
	case "push-phone":
		return outbound.LINEResult{}, s.operations.PushMessagesByPhone(ctx, c.Payload)
	case "push-template-phone":
		return outbound.LINEResult{}, s.operations.PushTemplatedMessagesByPhone(ctx, c.Payload)
	case "pnp-stats":
		return s.operations.PNPMessageStatistics(ctx, c.ID)
	case "pnp-template-stats":
		return s.operations.PNPTemplatedMessageStatistics(ctx, c.ID)
	case "create-coupon":
		return s.operations.CreateCoupon(ctx, c.Payload)
	case "close-coupon":
		return outbound.LINEResult{}, s.operations.CloseCoupon(ctx, c.ID)
	default:
		return outbound.LINEResult{}, errors.New("unsupported eligibility action")
	}
}
