// Package valueobjects defines immutable values used by the Audience context.
package valueobjects

import (
	"errors"
	"strings"
)

// AudienceID identifies one provider-independent audience definition.
type AudienceID string

func NewAudienceID(raw string) (AudienceID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("audience id is required")
	}
	return AudienceID(raw), nil
}

func (id AudienceID) String() string { return string(id) }
func (id AudienceID) Valid() bool    { return strings.TrimSpace(string(id)) != "" }

// Version identifies an immutable revision of an audience definition.
type Version uint64

const InitialVersion Version = 1

func (v Version) Valid() bool { return v > 0 }
func (v Version) Next() Version {
	return v + 1
}

// SelectionCriteria is an immutable, provider-independent selection definition.
// Source names the business source of the definition; Definition remains opaque
// until the Audience context owns a richer selection language.
type SelectionCriteria struct {
	source     string
	definition string
}

func NewSelectionCriteria(source, definition string) (SelectionCriteria, error) {
	source = strings.TrimSpace(source)
	definition = strings.TrimSpace(definition)
	if source == "" || definition == "" {
		return SelectionCriteria{}, errors.New("selection criteria source and definition are required")
	}
	return SelectionCriteria{source: source, definition: definition}, nil
}

func (c SelectionCriteria) Source() string     { return c.source }
func (c SelectionCriteria) Definition() string { return c.definition }
func (c SelectionCriteria) Valid() bool        { return c.source != "" && c.definition != "" }
func (c SelectionCriteria) Equal(other SelectionCriteria) bool {
	return c.source == other.source && c.definition == other.definition
}
