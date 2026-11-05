// Package valueobjects defines immutable presentation values.
package valueobjects

import (
	"errors"
	"strings"
)

type SurfaceID string

func NewSurfaceID(raw string) (SurfaceID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("surface id is required")
	}
	return SurfaceID(raw), nil
}

func (id SurfaceID) String() string { return string(id) }
func (id SurfaceID) Valid() bool    { return strings.TrimSpace(string(id)) != "" }

type Version uint64

const InitialVersion Version = 1

func (v Version) Valid() bool { return v > 0 }
func (v Version) Next() Version {
	return v + 1
}

// Region describes one independently actionable area of a presentation.
type Region struct {
	id              string
	actionReference string
	assetReference  string
}

func NewRegion(id, actionReference, assetReference string) (Region, error) {
	id = strings.TrimSpace(id)
	actionReference = strings.TrimSpace(actionReference)
	assetReference = strings.TrimSpace(assetReference)
	if id == "" || actionReference == "" {
		return Region{}, errors.New("region id and action reference are required")
	}
	return Region{id: id, actionReference: actionReference, assetReference: assetReference}, nil
}

func (r Region) ID() string              { return r.id }
func (r Region) ActionReference() string { return r.actionReference }
func (r Region) AssetReference() string  { return r.assetReference }
func (r Region) Valid() bool             { return r.id != "" && r.actionReference != "" }

// Presentation is an immutable, complete set of interaction regions.
type Presentation struct{ regions []Region }

func NewPresentation(regions []Region) (Presentation, error) {
	if len(regions) == 0 {
		return Presentation{}, errors.New("presentation requires at least one region")
	}
	seen := make(map[string]struct{}, len(regions))
	copyOfRegions := make([]Region, len(regions))
	for i, region := range regions {
		if !region.Valid() {
			return Presentation{}, errors.New("presentation contains an invalid region")
		}
		if _, exists := seen[region.ID()]; exists {
			return Presentation{}, errors.New("presentation contains duplicate region ids")
		}
		seen[region.ID()] = struct{}{}
		copyOfRegions[i] = region
	}
	return Presentation{regions: copyOfRegions}, nil
}

func (p Presentation) Valid() bool { return len(p.regions) > 0 }
func (p Presentation) Regions() []Region {
	return append([]Region(nil), p.regions...)
}

func (p Presentation) Equal(other Presentation) bool {
	if len(p.regions) != len(other.regions) {
		return false
	}
	for i := range p.regions {
		if p.regions[i] != other.regions[i] {
			return false
		}
	}
	return true
}
