package entities

import (
	"math"
	"strings"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type Workplace struct {
	id                valueobjects.WorkplaceID
	displayName       string
	active            bool
	center            valueobjects.GeoPoint
	radiusMeters      float64
	maxAccuracyMeters float64
	policyVersion     string
}

func DefineWorkplace(id valueobjects.WorkplaceID, displayName string, center valueobjects.GeoPoint, radiusMeters, maxAccuracyMeters float64, policyVersion string) (Workplace, error) {
	if !id.Valid() || strings.TrimSpace(displayName) == "" || math.IsNaN(radiusMeters) || math.IsInf(radiusMeters, 0) || radiusMeters <= 0 || math.IsNaN(maxAccuracyMeters) || math.IsInf(maxAccuracyMeters, 0) || maxAccuracyMeters <= 0 || strings.TrimSpace(policyVersion) == "" {
		return Workplace{}, ErrInvalidEntity
	}
	return Workplace{id: id, displayName: displayName, active: true, center: center, radiusMeters: radiusMeters, maxAccuracyMeters: maxAccuracyMeters, policyVersion: policyVersion}, nil
}

func (w Workplace) ID() valueobjects.WorkplaceID  { return w.id }
func (w Workplace) Center() valueobjects.GeoPoint { return w.center }
func (w Workplace) RadiusMeters() float64         { return w.radiusMeters }
func (w Workplace) MaxAccuracyMeters() float64    { return w.maxAccuracyMeters }
func (w Workplace) PolicyVersion() string         { return w.policyVersion }
func (w Workplace) DisplayName() string           { return w.displayName }
func (w Workplace) Active() bool                  { return w.active }
func (w *Workplace) Deactivate() error {
	if w == nil || !w.id.Valid() {
		return ErrInvalidEntity
	}
	w.active = false
	return nil
}

// AssessLocation converts short-lived raw evidence into the coordinate-free
// decision record retained by Attendance.
func (w Workplace) AssessLocation(evidence valueobjects.LocationEvidence, ref valueobjects.EvidenceRef) (valueobjects.LocationAssessment, error) {
	const earthRadiusMeters = 6371000.0
	lat1, lat2 := w.center.Latitude()*math.Pi/180, evidence.Latitude()*math.Pi/180
	dLat := (evidence.Latitude() - w.center.Latitude()) * math.Pi / 180
	dLon := (evidence.Longitude() - w.center.Longitude()) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	distance := earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return valueobjects.NewLocationAssessment(distance, evidence.AccuracyMeters(), distance > w.radiusMeters, evidence.AccuracyMeters() > w.maxAccuracyMeters, ref, w.policyVersion)
}
