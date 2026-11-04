// Package events defines credential-free facts about identity changes.
package events

import valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"

// IdentityEvent is a marker for identity domain events.
type IdentityEvent interface{ IdentityEvent() }

// AccountLinked is raised for a newly verified account relationship.
type AccountLinked struct{ ID valueobjects.ConnectionID }

func (AccountLinked) IdentityEvent() {}
