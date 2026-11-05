// Package events defines facts raised by the Audience aggregate.
package events

import valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"

type AudienceEvent interface{ AudienceEvent() }

type Defined struct {
	ID      valueobjects.AudienceID
	Version valueobjects.Version
}

func (Defined) AudienceEvent() {}

type Revised struct {
	ID      valueobjects.AudienceID
	Version valueobjects.Version
}

func (Revised) AudienceEvent() {}

type Activated struct {
	ID      valueobjects.AudienceID
	Version valueobjects.Version
}

func (Activated) AudienceEvent() {}

type Archived struct {
	ID      valueobjects.AudienceID
	Version valueobjects.Version
}

func (Archived) AudienceEvent() {}
