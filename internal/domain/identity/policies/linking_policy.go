// Package policies contains identity decisions that do not require I/O.
package policies

import (
	"errors"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// ValidateGrantedScopes requires each non-identity requested scope to be
// present among granted scopes. Google identity claims are verified upstream.
func ValidateGrantedScopes(requested valueobjects.OAuthScopes, granted []string) error {
	if !requested.Valid() {
		return errors.New("requested OAuth scopes missing")
	}
	grants := make(map[string]struct{}, len(granted))
	for _, scope := range granted {
		grants[scope] = struct{}{}
	}
	for _, scope := range requested.Values() {
		if scope == "openid" || scope == "email" {
			continue
		}
		if _, ok := grants[scope]; !ok {
			return errors.New("required Google authorization scope was not granted")
		}
	}
	return nil
}
