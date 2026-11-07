package valueobjects

import (
	"math"
	"testing"
)

func TestLocationFixValidation(t *testing.T) {
	value, err := NewLocationFix(25.03, 121.56, 10)
	if err != nil || !value.Valid() || value.Latitude() != 25.03 || value.Longitude() != 121.56 || value.AccuracyMeters() != 10 {
		t.Fatalf("valid fix = %+v, %v", value, err)
	}
	for _, input := range [][3]float64{{91, 0, 1}, {0, 181, 1}, {0, 0, 0}, {math.NaN(), 0, 1}} {
		if _, err := NewLocationFix(input[0], input[1], input[2]); err == nil {
			t.Fatalf("invalid fix %v accepted", input)
		}
	}
}
