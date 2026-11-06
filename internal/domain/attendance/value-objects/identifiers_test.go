package valueobjects

import "testing"

func TestAttendanceIdentifierContracts(t *testing.T) {
	tests := []struct {
		name    string
		create  func(string) (string, error)
		valid   func(string) bool
		errText string
	}{
		{"participant", func(v string) (string, error) { x, err := NewParticipantID(v); return string(x), err }, func(v string) bool { return ParticipantID(v).Valid() }, "participant ID is required"},
		{"workplace", func(v string) (string, error) { x, err := NewWorkplaceID(v); return string(x), err }, func(v string) bool { return WorkplaceID(v).Valid() }, "workplace ID is required"},
		{"shift instance", func(v string) (string, error) { x, err := NewShiftInstanceID(v); return string(x), err }, func(v string) bool { return ShiftInstanceID(v).Valid() }, "shift instance ID is required"},
		{"shift assignment", func(v string) (string, error) { x, err := NewShiftAssignmentID(v); return string(x), err }, func(v string) bool { return ShiftAssignmentID(v).Valid() }, "shift assignment ID is required"},
		{"submission", func(v string) (string, error) { x, err := NewSubmissionID(v); return string(x), err }, func(v string) bool { return SubmissionID(v).Valid() }, "submission ID is required"},
		{"enrollment code hash", func(v string) (string, error) { x, err := NewEnrollmentCodeHash(v); return string(x), err }, func(v string) bool { return EnrollmentCodeHash(v).Valid() }, "enrollment code hash is required"},
		{"evidence reference", func(v string) (string, error) { x, err := NewEvidenceRef(v); return string(x), err }, func(v string) bool { return EvidenceRef(v).Valid() }, "evidence reference is required"},
		{"employee", func(v string) (string, error) { x, err := NewEmployeeID(v); return string(x), err }, func(v string) bool { return EmployeeID(v).Valid() }, "employee ID is required"},
		{"punch", func(v string) (string, error) { x, err := NewPunchID(v); return string(x), err }, func(v string) bool { return PunchID(v).Valid() }, "punch ID is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.create(" \t "); err == nil || err.Error() != test.errText {
				t.Fatalf("whitespace error = %v", err)
			}
			if test.valid(" \t ") {
				t.Fatal("whitespace-only identifier is valid")
			}
			const value = " preserved "
			got, err := test.create(value)
			if err != nil || got != value || !test.valid(got) {
				t.Fatalf("identifier = %q, valid = %t, err = %v", got, test.valid(got), err)
			}
		})
	}
}
