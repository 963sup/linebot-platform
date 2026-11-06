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

func NewParticipantID(value string) (ParticipantID, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("participant ID is required")
	}
	return ParticipantID(value), nil
}
func NewWorkplaceID(value string) (WorkplaceID, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("workplace ID is required")
	}
	return WorkplaceID(value), nil
}
func NewShiftInstanceID(value string) (ShiftInstanceID, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("shift instance ID is required")
	}
	return ShiftInstanceID(value), nil
}
func NewShiftAssignmentID(value string) (ShiftAssignmentID, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("shift assignment ID is required")
	}
	return ShiftAssignmentID(value), nil
}
func NewSubmissionID(value string) (SubmissionID, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("submission ID is required")
	}
	return SubmissionID(value), nil
}
func NewEnrollmentCodeHash(value string) (EnrollmentCodeHash, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("enrollment code hash is required")
	}
	return EnrollmentCodeHash(value), nil
}
func NewEvidenceRef(value string) (EvidenceRef, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("evidence reference is required")
	}
	return EvidenceRef(value), nil
}
func NewEmployeeID(value string) (EmployeeID, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("employee ID is required")
	}
	return EmployeeID(value), nil
}
func NewPunchID(value string) (PunchID, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("punch ID is required")
	}
	return PunchID(value), nil
}

func (v ParticipantID) Valid() bool      { return strings.TrimSpace(string(v)) != "" }
func (v WorkplaceID) Valid() bool        { return strings.TrimSpace(string(v)) != "" }
func (v ShiftInstanceID) Valid() bool    { return strings.TrimSpace(string(v)) != "" }
func (v ShiftAssignmentID) Valid() bool  { return strings.TrimSpace(string(v)) != "" }
func (v SubmissionID) Valid() bool       { return strings.TrimSpace(string(v)) != "" }
func (v EnrollmentCodeHash) Valid() bool { return strings.TrimSpace(string(v)) != "" }
func (v EvidenceRef) Valid() bool        { return strings.TrimSpace(string(v)) != "" }
func (v EmployeeID) Valid() bool         { return strings.TrimSpace(string(v)) != "" }
func (v PunchID) Valid() bool            { return strings.TrimSpace(string(v)) != "" }
