// Package valueobjects defines immutable workspace automation values.
package valueobjects

import (
	"errors"
	"strings"
)

// AutomationRequestID identifies one automation request.
type AutomationRequestID struct{ value string }

func NewAutomationRequestID(value string) (AutomationRequestID, error) {
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\t\r\n") {
		return AutomationRequestID{}, errors.New("invalid automation request ID")
	}
	return AutomationRequestID{value: value}, nil
}

func (id AutomationRequestID) String() string { return id.value }
func (id AutomationRequestID) Valid() bool    { return id.value != "" }
