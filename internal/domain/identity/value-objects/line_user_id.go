package valueobjects

import (
	"errors"
	"strings"
)

// LineUserID is a validated identifier, not proof of caller authentication.
type LineUserID struct{ value string }

func NewLineUserID(value string) (LineUserID, error) {
	if value == "" || value != strings.TrimSpace(value) {
		return LineUserID{}, errors.New("invalid LINE user ID")
	}
	return LineUserID{value: value}, nil
}

func (id LineUserID) String() string { return id.value }
func (id LineUserID) Valid() bool   { return id.value != "" }
