// Package aggregates owns Experience Presentation consistency boundaries.
package aggregates

import (
	"errors"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
)

var (
	ErrInvalidInteractionSurface = errors.New("invalid interaction surface")
	ErrInvalidSurfaceTransition  = errors.New("invalid interaction surface lifecycle transition")
	ErrPublishedSurfaceImmutable = errors.New("published interaction surface is immutable")
)

type Status uint8

const (
	StatusDraft Status = iota + 1
	StatusPublished
	StatusRetired
)

func (s Status) Valid() bool { return s >= StatusDraft && s <= StatusRetired }

// InteractionSurface owns one versioned desired presentation. Publishing
// freezes that version; further changes require a new aggregate/version line.
type InteractionSurface struct {
	id           valueobjects.SurfaceID
	presentation valueobjects.Presentation
	version      valueobjects.Version
	status       Status
}

func NewInteractionSurface(id valueobjects.SurfaceID, presentation valueobjects.Presentation) (InteractionSurface, error) {
	if !id.Valid() || !presentation.Valid() {
		return InteractionSurface{}, ErrInvalidInteractionSurface
	}
	return InteractionSurface{
		id:           id,
		presentation: presentation,
		version:      valueobjects.InitialVersion,
		status:       StatusDraft,
	}, nil
}

// RestoreInteractionSurface rehydrates persisted state.
func RestoreInteractionSurface(id valueobjects.SurfaceID, presentation valueobjects.Presentation, version valueobjects.Version, status Status) (InteractionSurface, error) {
	if !id.Valid() || !presentation.Valid() || !version.Valid() || !status.Valid() {
		return InteractionSurface{}, ErrInvalidInteractionSurface
	}
	return InteractionSurface{id: id, presentation: presentation, version: version, status: status}, nil
}

func (s InteractionSurface) ID() valueobjects.SurfaceID              { return s.id }
func (s InteractionSurface) Presentation() valueobjects.Presentation { return s.presentation }
func (s InteractionSurface) Version() valueobjects.Version           { return s.version }
func (s InteractionSurface) Status() Status                          { return s.status }

func (s *InteractionSurface) Revise(presentation valueobjects.Presentation) error {
	if s == nil || !presentation.Valid() {
		return ErrInvalidInteractionSurface
	}
	if s.status == StatusPublished || s.status == StatusRetired {
		return ErrPublishedSurfaceImmutable
	}
	if s.status != StatusDraft {
		return ErrInvalidInteractionSurface
	}
	if s.presentation.Equal(presentation) {
		return nil
	}
	s.presentation = presentation
	s.version = s.version.Next()
	return nil
}

func (s *InteractionSurface) Publish() error {
	if s == nil || s.status != StatusDraft || !s.presentation.Valid() {
		return ErrInvalidSurfaceTransition
	}
	s.status = StatusPublished
	s.version = s.version.Next()
	return nil
}

func (s *InteractionSurface) Retire() error {
	if s == nil || s.status != StatusPublished {
		return ErrInvalidSurfaceTransition
	}
	s.status = StatusRetired
	s.version = s.version.Next()
	return nil
}
