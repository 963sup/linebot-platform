package valueobjects

import (
	"errors"
	"strings"
)

// Approval records the principal that approved an automation request.
type Approval struct{ approvedBy string }

func NewApproval(approvedBy string) (Approval, error) {
	if approvedBy == "" || approvedBy != strings.TrimSpace(approvedBy) || strings.ContainsAny(approvedBy, "\r\n\t") {
		return Approval{}, errors.New("invalid approval principal")
	}
	return Approval{approvedBy: approvedBy}, nil
}

func (a Approval) ApprovedBy() string { return a.approvedBy }
func (a Approval) Valid() bool        { return a.approvedBy != "" }
