// Package specifications holds reusable predicates over identity models.
package specifications

import "github.com/963sup/linebot-platform/internal/domain/identity/aggregates"

// CanUseGoogleAPI checks active status and exact required OAuth scopes.
// Caller authentication and ownership must be checked separately.
func CanUseGoogleAPI(link aggregates.AccountLink, required []string) bool {
	if !link.IsActive() {
		return false
	}
	for _, scope := range required {
		if scope == "" || !link.HasScope(scope) {
			return false
		}
	}
	return true
}
