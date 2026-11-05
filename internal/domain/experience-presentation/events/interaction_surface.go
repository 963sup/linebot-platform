// Package events defines facts raised by InteractionSurface.
package events

import valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"

type SurfaceEvent interface{ SurfaceEvent() }

type Defined struct {
	ID      valueobjects.SurfaceID
	Version valueobjects.Version
}

func (Defined) SurfaceEvent() {}

type Revised struct {
	ID      valueobjects.SurfaceID
	Version valueobjects.Version
}

func (Revised) SurfaceEvent() {}

type Published struct {
	ID      valueobjects.SurfaceID
	Version valueobjects.Version
}

func (Published) SurfaceEvent() {}

type Retired struct {
	ID      valueobjects.SurfaceID
	Version valueobjects.Version
}

func (Retired) SurfaceEvent() {}
