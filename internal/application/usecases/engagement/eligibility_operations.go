package engagement

import (
	"errors"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// EligibilityOperations is the application boundary for PNP and coupon APIs.
type EligibilityOperations struct {
	operations outbound.EligibilityOperations
}

func NewEligibilityOperations(o outbound.EligibilityOperations) (*EligibilityOperations, error) {
	if o == nil {
		return nil, errors.New("eligibility operations are required")
	}
	return &EligibilityOperations{operations: o}, nil
}
