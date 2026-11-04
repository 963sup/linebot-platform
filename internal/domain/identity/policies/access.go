// Package policies contains pure identity authorization decisions.
package policies

import "github.com/963sup/linebot-platform/internal/domain/identity/aggregates"

// CanUseGoogleAPI checks link status and granted scopes.
// The caller and resource owner must additionally be authenticated by the application.
func CanUseGoogleAPI(link aggregates.AccountLink, required []string) bool {
	if !link.IsActive() {
		return false
	}
	granted := make(map[string]struct{}, len(link.Scopes()))
	for _, scope := range link.Scopes() {
		granted[scope] = struct{}{}
	}
	for _, scope := range required {
		if scope == "" {
			return false
		}
		if _, ok := granted[scope]; !ok {
			return false
		}
	}
	return true
}
