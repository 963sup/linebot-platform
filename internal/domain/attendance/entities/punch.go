package entities

import (
	"errors"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var ErrInvalidPunch = errors.New("invalid attendance punch")

type Punch struct {
	id         valueobjects.PunchID
	kind       valueobjects.PunchKind
	recordedAt time.Time
	location   valueobjects.LocationFix
}

func NewPunch(id valueobjects.PunchID, kind valueobjects.PunchKind, recordedAt time.Time, location valueobjects.LocationFix) (Punch, error) {
	if !id.Valid() || !kind.Valid() || recordedAt.IsZero() || !location.Valid() {
		return Punch{}, ErrInvalidPunch
	}
	return Punch{id: id, kind: kind, recordedAt: recordedAt.UTC(), location: location}, nil
}

func RestorePunch(id valueobjects.PunchID, kind valueobjects.PunchKind, recordedAt time.Time, location valueobjects.LocationFix) (Punch, error) {
	return NewPunch(id, kind, recordedAt, location)
}

func (p Punch) ID() valueobjects.PunchID           { return p.id }
func (p Punch) Kind() valueobjects.PunchKind       { return p.kind }
func (p Punch) RecordedAt() time.Time              { return p.recordedAt }
func (p Punch) Location() valueobjects.LocationFix { return p.location }
