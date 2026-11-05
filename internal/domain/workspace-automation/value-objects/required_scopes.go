package valueobjects

import (
	"errors"
	"sort"
	"strings"
)

// RequiredScopes is an immutable canonical set of scopes needed by an action.
type RequiredScopes struct{ values []string }

func NewRequiredScopes(scopes []string) (RequiredScopes, error) {
	if len(scopes) == 0 {
		return RequiredScopes{}, errors.New("required scopes cannot be empty")
	}
	unique := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if scope == "" || scope != strings.TrimSpace(scope) || strings.ContainsAny(scope, " \t\r\n") {
			return RequiredScopes{}, errors.New("invalid required scope")
		}
		unique[scope] = struct{}{}
	}
	values := make([]string, 0, len(unique))
	for scope := range unique {
		values = append(values, scope)
	}
	sort.Strings(values)
	return RequiredScopes{values: values}, nil
}

func (s RequiredScopes) Values() []string { return append([]string(nil), s.values...) }
func (s RequiredScopes) Valid() bool      { return len(s.values) > 0 }
func (s RequiredScopes) Contains(scope string) bool {
	i := sort.SearchStrings(s.values, scope)
	return i < len(s.values) && s.values[i] == scope
}
