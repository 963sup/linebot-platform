package valueobjects

import "testing"

func TestIdentifiersRejectBlank(t *testing.T) {
	constructors := []func(string) error{
		func(v string) error { _, err := NewParticipantID(v); return err },
		func(v string) error { _, err := NewAttendanceDayID(v); return err },
		func(v string) error { _, err := NewPunchID(v); return err },
	}
	for _, constructor := range constructors {
		if err := constructor(" "); err == nil {
			t.Fatal("blank identifier accepted")
		}
	}
}
