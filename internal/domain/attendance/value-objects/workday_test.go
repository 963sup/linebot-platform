package valueobjects

import (
	"testing"
	"time"
)

func TestWorkdayAtUsesBusinessTimezone(t *testing.T) {
	taipei := time.FixedZone("Asia/Taipei", 8*60*60)
	got, err := WorkdayAt(time.Date(2026, 11, 6, 16, 30, 0, 0, time.UTC), taipei)
	if err != nil || got != "2026-11-07" {
		t.Fatalf("WorkdayAt() = %q, %v", got, err)
	}
	for _, invalid := range []string{"", "2026-1-1", "2026-02-30"} {
		if _, err := NewWorkday(invalid); err == nil {
			t.Fatalf("NewWorkday(%q) succeeded", invalid)
		}
	}
}
