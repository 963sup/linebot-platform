package entities

import (
	"testing"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func TestWorkplaceAssessesRawEvidenceWithoutRetainingCoordinates(t *testing.T) {
	id, _ := valueobjects.NewWorkplaceID("workplace-1")
	center, _ := valueobjects.NewGeoPoint(25, 121)
	workplace, _ := DefineWorkplace(id, "Office", center, 100, 50, "policy-v1")
	evidence, _ := valueobjects.NewLocationEvidence(25.01, 121.01, 80)
	ref, _ := valueobjects.NewEvidenceRef("evidence/punch-1")
	assessment, err := workplace.AssessLocation(evidence, ref)
	if err != nil {
		t.Fatal(err)
	}
	if !assessment.OutsideWorkplace() || !assessment.LowAccuracy() || assessment.PolicyVersion() != "policy-v1" {
		t.Fatalf("assessment: %+v", assessment)
	}
}
