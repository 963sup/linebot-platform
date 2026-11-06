package valueobjects

import "testing"

func TestLocationEvidenceRequiresPositiveAccuracy(t *testing.T) {
	if _, err := NewLocationEvidence(25, 121, 0); err == nil {
		t.Fatal("zero accuracy accepted")
	}
	if _, err := NewLocationEvidence(25, 121, 1); err != nil {
		t.Fatal(err)
	}
}
