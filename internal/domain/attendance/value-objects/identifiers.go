// Package valueobjects defines immutable Attendance values.
package valueobjects

import (
	"errors"
	"strings"
)

type ParticipantID string
type AttendanceDayID string
type EnrollmentCodeHash string
type EmployeeID string
type PunchID string

type identifier interface{ ~string }

func newIdentifier[T identifier](value, label string) (T, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New(label + " is required")
	}
	return T(value), nil
}

func validIdentifier[T identifier](value T) bool { return strings.TrimSpace(string(value)) != "" }

func NewParticipantID(value string) (ParticipantID, error) {
	return newIdentifier[ParticipantID](value, "participant ID")
}
func NewAttendanceDayID(value string) (AttendanceDayID, error) {
	return newIdentifier[AttendanceDayID](value, "attendance day ID")
}
func NewEnrollmentCodeHash(value string) (EnrollmentCodeHash, error) {
	return newIdentifier[EnrollmentCodeHash](value, "enrollment code hash")
}
func NewEmployeeID(value string) (EmployeeID, error) {
	return newIdentifier[EmployeeID](value, "employee ID")
}
func NewPunchID(value string) (PunchID, error) { return newIdentifier[PunchID](value, "punch ID") }

func (v ParticipantID) Valid() bool      { return validIdentifier(v) }
func (v AttendanceDayID) Valid() bool    { return validIdentifier(v) }
func (v EnrollmentCodeHash) Valid() bool { return validIdentifier(v) }
func (v EmployeeID) Valid() bool         { return validIdentifier(v) }
func (v PunchID) Valid() bool            { return validIdentifier(v) }
