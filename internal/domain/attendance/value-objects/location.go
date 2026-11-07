package valueobjects

import (
	"errors"
	"math"
)

// LocationFix is the browser-provided GPS fix recorded with one Punch.
type LocationFix struct{ latitude, longitude, accuracyMeters float64 }

func NewLocationFix(latitude, longitude, accuracyMeters float64) (LocationFix, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 ||
		math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 ||
		math.IsNaN(accuracyMeters) || math.IsInf(accuracyMeters, 0) || accuracyMeters <= 0 {
		return LocationFix{}, errors.New("invalid location fix")
	}
	return LocationFix{latitude: latitude, longitude: longitude, accuracyMeters: accuracyMeters}, nil
}

func (v LocationFix) Latitude() float64       { return v.latitude }
func (v LocationFix) Longitude() float64      { return v.longitude }
func (v LocationFix) AccuracyMeters() float64 { return v.accuracyMeters }
func (v LocationFix) Valid() bool {
	_, err := NewLocationFix(v.latitude, v.longitude, v.accuracyMeters)
	return err == nil
}
func (v LocationFix) Equal(other LocationFix) bool { return v == other }
