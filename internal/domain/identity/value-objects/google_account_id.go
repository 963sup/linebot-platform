package valueobjects

import (
	"errors"
	"strings"
)

// GoogleAccountID stores the verified Google subject (sub), never an email.
type GoogleAccountID struct{ value string }

func NewGoogleAccountID(value string) (GoogleAccountID, error) {
	if value == "" || value != strings.TrimSpace(value) {
		return GoogleAccountID{}, errors.New("invalid Google account ID")
	}
	return GoogleAccountID{value: value}, nil
}

func (id GoogleAccountID) String() string { return id.value }
func (id GoogleAccountID) Valid() bool   { return id.value != "" }
