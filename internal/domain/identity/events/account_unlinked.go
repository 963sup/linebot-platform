package events

import valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"

// AccountUnlinked represents a completed domain state transition.
// Publishing is handled only after the persistence transaction commits.
type AccountUnlinked struct{ ID valueobjects.ConnectionID }

func (AccountUnlinked) IdentityEvent() {}
