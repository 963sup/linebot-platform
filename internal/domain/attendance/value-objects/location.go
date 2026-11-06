package valueobjects

import (
	"errors"
	"math"
	"strings"
)

// LocationEvidence is short-lived raw input and must not be stored in Attendance.
type LocationEvidence struct{ latitude, longitude, accuracyMeters float64 }

func NewLocationEvidence(latitude, longitude, accuracyMeters float64) (LocationEvidence, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 || math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 || math.IsNaN(accuracyMeters) || math.IsInf(accuracyMeters, 0) || accuracyMeters <= 0 {
		return LocationEvidence{}, errors.New("invalid location evidence")
	}
	return LocationEvidence{latitude: latitude, longitude: longitude, accuracyMeters: accuracyMeters}, nil
}
func (v LocationEvidence) Latitude() float64       { return v.latitude }
func (v LocationEvidence) Longitude() float64      { return v.longitude }
func (v LocationEvidence) AccuracyMeters() float64 { return v.accuracyMeters }

type GeoPoint struct{ latitude, longitude float64 }

func NewGeoPoint(latitude, longitude float64) (GeoPoint, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 || math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return GeoPoint{}, errors.New("invalid geographic point")
	}
	return GeoPoint{latitude: latitude, longitude: longitude}, nil
}
func (v GeoPoint) Latitude() float64  { return v.latitude }
func (v GeoPoint) Longitude() float64 { return v.longitude }

// LocationAssessment is the long-lived decision record. It deliberately has no coordinates.
type LocationAssessment struct {
	distanceMeters   float64
	accuracyMeters   float64
	outsideWorkplace bool
	lowAccuracy      bool
	evidenceRef      EvidenceRef
	policyVersion    string
}

func NewLocationAssessment(distanceMeters, accuracyMeters float64, outsideWorkplace, lowAccuracy bool, evidenceRef EvidenceRef, policyVersion string) (LocationAssessment, error) {
	if math.IsNaN(distanceMeters) || math.IsInf(distanceMeters, 0) || distanceMeters < 0 || math.IsNaN(accuracyMeters) || math.IsInf(accuracyMeters, 0) || accuracyMeters <= 0 || !evidenceRef.Valid() || strings.TrimSpace(policyVersion) == "" {
		return LocationAssessment{}, errors.New("invalid location assessment")
	}
	return LocationAssessment{distanceMeters: distanceMeters, accuracyMeters: accuracyMeters, outsideWorkplace: outsideWorkplace, lowAccuracy: lowAccuracy, evidenceRef: evidenceRef, policyVersion: policyVersion}, nil
}
func (v LocationAssessment) DistanceMeters() float64  { return v.distanceMeters }
func (v LocationAssessment) AccuracyMeters() float64  { return v.accuracyMeters }
func (v LocationAssessment) OutsideWorkplace() bool   { return v.outsideWorkplace }
func (v LocationAssessment) LowAccuracy() bool        { return v.lowAccuracy }
func (v LocationAssessment) EvidenceRef() EvidenceRef { return v.evidenceRef }
func (v LocationAssessment) PolicyVersion() string    { return v.policyVersion }
func (v LocationAssessment) RequiresReview() bool     { return v.outsideWorkplace || v.lowAccuracy }
func (v LocationAssessment) Valid() bool {
	_, err := NewLocationAssessment(v.distanceMeters, v.accuracyMeters, v.outsideWorkplace, v.lowAccuracy, v.evidenceRef, v.policyVersion)
	return err == nil
}
func (v LocationAssessment) Equal(other LocationAssessment) bool { return v == other }
