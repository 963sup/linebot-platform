// Package aggregates owns Audience consistency boundaries.
package aggregates

import (
	"errors"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
)

var (
	ErrInvalidAudienceDefinition = errors.New("invalid audience definition")
	ErrInvalidAudienceTransition = errors.New("invalid audience lifecycle transition")
	ErrArchivedAudience          = errors.New("archived audience cannot be changed")
)

type Status uint8

const (
	StatusDraft Status = iota + 1
	StatusActive
	StatusArchived
)

func (s Status) Valid() bool { return s >= StatusDraft && s <= StatusArchived }

// AudienceDefinition owns provider-independent criteria and lifecycle.
type AudienceDefinition struct {
	id       valueobjects.AudienceID
	criteria valueobjects.SelectionCriteria
	version  valueobjects.Version
	status   Status
}

func NewAudienceDefinition(id valueobjects.AudienceID, criteria valueobjects.SelectionCriteria) (AudienceDefinition, error) {
	if !id.Valid() || !criteria.Valid() {
		return AudienceDefinition{}, ErrInvalidAudienceDefinition
	}
	return AudienceDefinition{
		id:       id,
		criteria: criteria,
		version:  valueobjects.InitialVersion,
		status:   StatusDraft,
	}, nil
}

// RestoreAudienceDefinition rehydrates persisted state.
func RestoreAudienceDefinition(id valueobjects.AudienceID, criteria valueobjects.SelectionCriteria, version valueobjects.Version, status Status) (AudienceDefinition, error) {
	if !id.Valid() || !criteria.Valid() || !version.Valid() || !status.Valid() {
		return AudienceDefinition{}, ErrInvalidAudienceDefinition
	}
	return AudienceDefinition{id: id, criteria: criteria, version: version, status: status}, nil
}

func (a AudienceDefinition) ID() valueobjects.AudienceID              { return a.id }
func (a AudienceDefinition) Criteria() valueobjects.SelectionCriteria { return a.criteria }
func (a AudienceDefinition) Version() valueobjects.Version            { return a.version }
func (a AudienceDefinition) Status() Status                           { return a.status }

// Revise replaces the definition and advances its immutable version. Active
// audiences remain active; archived audiences cannot be revised.
func (a *AudienceDefinition) Revise(criteria valueobjects.SelectionCriteria) error {
	if a == nil || !criteria.Valid() {
		return ErrInvalidAudienceDefinition
	}
	if a.status == StatusArchived {
		return ErrArchivedAudience
	}
	if !a.status.Valid() {
		return ErrInvalidAudienceDefinition
	}
	if a.criteria.Equal(criteria) {
		return nil
	}
	a.criteria = criteria
	a.version = a.version.Next()
	return nil
}

func (a *AudienceDefinition) Activate() error {
	if a == nil || a.status != StatusDraft {
		return ErrInvalidAudienceTransition
	}
	a.status = StatusActive
	a.version = a.version.Next()
	return nil
}

func (a *AudienceDefinition) Archive() error {
	if a == nil || a.status == StatusArchived || !a.status.Valid() {
		return ErrInvalidAudienceTransition
	}
	a.status = StatusArchived
	a.version = a.version.Next()
	return nil
}
