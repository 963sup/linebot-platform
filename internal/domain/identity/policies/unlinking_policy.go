package policies

import (
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

// ValidateUnlink ensures the authenticated caller targets the active link.
// Authentication itself must happen before this policy is called.
func ValidateUnlink(link aggregates.AccountLink, owner valueobjects.LineUserID, subject valueobjects.GoogleAccountID) error {
	if !owner.Valid() || !subject.Valid() || !link.IsActive() ||
		link.Owner() != owner || link.GoogleAccount() != subject {
		return errors.New("account link is not eligible for unlinking")
	}
	return nil
}
