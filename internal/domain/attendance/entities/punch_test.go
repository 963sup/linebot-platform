package entities

import (
	"testing"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func TestPunchRequiresServerTimeAndLocation(t *testing.T) {
	id, _ := valueobjects.NewPunchID("punch")
	fix, _ := valueobjects.NewLocationFix(25, 121, 8)
	when := time.Date(2026, 11, 7, 1, 2, 3, 0, time.FixedZone("source", 3600))
	punch, err := NewPunch(id, valueobjects.ClockIn, when, fix)
	if err != nil || punch.RecordedAt().Location() != time.UTC || !punch.Location().Equal(fix) {
		t.Fatalf("NewPunch() = %+v, %v", punch, err)
	}
	if _, err := NewPunch(id, valueobjects.ClockIn, time.Time{}, fix); err == nil {
		t.Fatal("zero server time accepted")
	}
}
