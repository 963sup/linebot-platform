package valueobjects

import (
	"errors"
	"regexp"
)

var identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

type TenantID string
type ChannelConnectionID string

func NewTenantID(value string) (TenantID, error) {
	if !identifierPattern.MatchString(value) {
		return "", errors.New("valid tenant ID is required")
	}
	return TenantID(value), nil
}

func NewChannelConnectionID(value string) (ChannelConnectionID, error) {
	if !identifierPattern.MatchString(value) {
		return "", errors.New("valid channel connection ID is required")
	}
	return ChannelConnectionID(value), nil
}

func (id TenantID) String() string            { return string(id) }
func (id ChannelConnectionID) String() string { return string(id) }
func (id TenantID) Valid() bool               { return identifierPattern.MatchString(string(id)) }
func (id ChannelConnectionID) Valid() bool    { return identifierPattern.MatchString(string(id)) }
