package entities

import (
	"errors"
	"strings"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var ErrEnrollmentConflict = errors.New("attendance enrollment binding conflicts")

// Participant is the Attendance-owned employee projection. It may be unbound
// until an invitation is atomically consumed by a verified participant.
type Participant struct {
	employeeID    valueobjects.EmployeeID
	employeeCode  string
	displayName   string
	active        bool
	participantID valueobjects.ParticipantID
	deactivatedAt time.Time
}

func NewParticipant(employeeID valueobjects.EmployeeID, employeeCode, displayName string) (Participant, error) {
	if !employeeID.Valid() || strings.TrimSpace(employeeCode) == "" || strings.TrimSpace(displayName) == "" {
		return Participant{}, ErrInvalidEntity
	}
	return Participant{employeeID: employeeID, employeeCode: employeeCode, displayName: displayName, active: true}, nil
}
func RestoreParticipant(employeeID valueobjects.EmployeeID, employeeCode, displayName string, active bool, participantID valueobjects.ParticipantID, deactivatedAt time.Time) (Participant, error) {
	if !employeeID.Valid() || strings.TrimSpace(employeeCode) == "" || strings.TrimSpace(displayName) == "" || active && !deactivatedAt.IsZero() || !active && deactivatedAt.IsZero() {
		return Participant{}, ErrInvalidEntity
	}
	return Participant{employeeID: employeeID, employeeCode: employeeCode, displayName: displayName, active: active, participantID: participantID, deactivatedAt: deactivatedAt.UTC()}, nil
}
func (p Participant) EmployeeID() valueobjects.EmployeeID { return p.employeeID }
func (p Participant) EmployeeCode() string                { return p.employeeCode }
func (p Participant) DisplayName() string                 { return p.displayName }
func (p Participant) Active() bool                        { return p.active }
func (p Participant) ParticipantID() (valueobjects.ParticipantID, bool) {
	return p.participantID, p.participantID.Valid()
}
func (p Participant) DeactivatedAt() time.Time { return p.deactivatedAt }
func (p *Participant) Bind(participantID valueobjects.ParticipantID) (valueobjects.EnrollmentOutcome, error) {
	if p == nil || !p.active || !participantID.Valid() {
		return "", ErrInvalidEntity
	}
	if p.participantID.Valid() && p.participantID != participantID {
		return "", ErrEnrollmentConflict
	}
	if p.participantID == participantID {
		return valueobjects.EnrollmentAlreadyBound, nil
	}
	p.participantID = participantID
	return valueobjects.EnrollmentBound, nil
}
func (p *Participant) Deactivate(serverTime time.Time) error {
	if p == nil || !p.employeeID.Valid() || serverTime.IsZero() {
		return ErrInvalidEntity
	}
	if p.active {
		p.active = false
		p.deactivatedAt = serverTime.UTC()
	}
	return nil
}

type EnrollmentInvitation struct {
	employeeID valueobjects.EmployeeID
	codeHash   valueobjects.EnrollmentCodeHash
	expiresAt  time.Time
}

func NewEnrollmentInvitation(employeeID valueobjects.EmployeeID, codeHash valueobjects.EnrollmentCodeHash, expiresAt time.Time) (EnrollmentInvitation, error) {
	if !employeeID.Valid() || !codeHash.Valid() || expiresAt.IsZero() {
		return EnrollmentInvitation{}, ErrInvalidEntity
	}
	return EnrollmentInvitation{employeeID: employeeID, codeHash: codeHash, expiresAt: expiresAt.UTC()}, nil
}
func (i EnrollmentInvitation) EmployeeID() valueobjects.EmployeeID       { return i.employeeID }
func (i EnrollmentInvitation) CodeHash() valueobjects.EnrollmentCodeHash { return i.codeHash }
func (i EnrollmentInvitation) ExpiresAt() time.Time                      { return i.expiresAt }
