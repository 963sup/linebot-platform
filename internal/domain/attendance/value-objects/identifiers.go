// Package valueobjects defines immutable Attendance values.
package valueobjects

import (
	"errors"
	"strings"
)

type ParticipantID string
type WorkplaceID string
type ShiftInstanceID string
type ShiftAssignmentID string
type SubmissionID string
type EnrollmentCodeHash string
type EvidenceRef string
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
func NewWorkplaceID(value string) (WorkplaceID, error) {
	return newIdentifier[WorkplaceID](value, "workplace ID")
}
func NewShiftInstanceID(value string) (ShiftInstanceID, error) {
	return newIdentifier[ShiftInstanceID](value, "shift instance ID")
}
func NewShiftAssignmentID(value string) (ShiftAssignmentID, error) {
	return newIdentifier[ShiftAssignmentID](value, "shift assignment ID")
}
func NewSubmissionID(value string) (SubmissionID, error) {
	return newIdentifier[SubmissionID](value, "submission ID")
}
func NewEnrollmentCodeHash(value string) (EnrollmentCodeHash, error) {
	return newIdentifier[EnrollmentCodeHash](value, "enrollment code hash")
}
func NewEvidenceRef(value string) (EvidenceRef, error) {
	return newIdentifier[EvidenceRef](value, "evidence reference")
}
func NewEmployeeID(value string) (EmployeeID, error) {
	return newIdentifier[EmployeeID](value, "employee ID")
}
func NewPunchID(value string) (PunchID, error) {
	return newIdentifier[PunchID](value, "punch ID")
}

func (v ParticipantID) Valid() bool      { return validIdentifier(v) }
func (v WorkplaceID) Valid() bool        { return validIdentifier(v) }
func (v ShiftInstanceID) Valid() bool    { return validIdentifier(v) }
func (v ShiftAssignmentID) Valid() bool  { return validIdentifier(v) }
func (v SubmissionID) Valid() bool       { return validIdentifier(v) }
func (v EnrollmentCodeHash) Valid() bool { return validIdentifier(v) }
func (v EvidenceRef) Valid() bool        { return validIdentifier(v) }
func (v EmployeeID) Valid() bool         { return validIdentifier(v) }
func (v PunchID) Valid() bool            { return validIdentifier(v) }
