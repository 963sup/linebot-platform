package firestore

import (
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type attendanceParticipantDocument struct {
	SchemaVersion int        `firestore:"schema_version"`
	EmployeeID    string     `firestore:"employee_id"`
	EmployeeCode  string     `firestore:"employee_code"`
	DisplayName   string     `firestore:"display_name"`
	Active        bool       `firestore:"active"`
	ParticipantID string     `firestore:"participant_id,omitempty"`
	CreatedAt     time.Time  `firestore:"created_at"`
	UpdatedAt     time.Time  `firestore:"updated_at"`
	DeactivatedAt *time.Time `firestore:"deactivated_at,omitempty"`
}

type attendanceEnrollmentDocument struct {
	SchemaVersion int        `firestore:"schema_version"`
	CodeHash      string     `firestore:"code_hash"`
	EmployeeID    string     `firestore:"employee_id"`
	ExpiresAt     time.Time  `firestore:"expires_at"`
	CreatedAt     time.Time  `firestore:"created_at"`
	ConsumedAt    *time.Time `firestore:"consumed_at,omitempty"`
	ParticipantID string     `firestore:"participant_id,omitempty"`
}

type attendanceBindingDocument struct {
	SchemaVersion int       `firestore:"schema_version"`
	ParticipantID string    `firestore:"participant_id"`
	EmployeeID    string    `firestore:"employee_id"`
	CreatedAt     time.Time `firestore:"created_at"`
}

type attendancePunchDocument struct {
	ID             string    `firestore:"id"`
	Kind           string    `firestore:"kind"`
	RecordedAt     time.Time `firestore:"recorded_at"`
	Latitude       float64   `firestore:"latitude"`
	Longitude      float64   `firestore:"longitude"`
	AccuracyMeters float64   `firestore:"accuracy_meters"`
}

type attendanceDayDocument struct {
	SchemaVersion int                      `firestore:"schema_version"`
	ID            string                   `firestore:"id"`
	ParticipantID string                   `firestore:"participant_id"`
	Workday       string                   `firestore:"workday"`
	Revision      int64                    `firestore:"revision"`
	ClockIn       *attendancePunchDocument `firestore:"clock_in,omitempty"`
	ClockOut      *attendancePunchDocument `firestore:"clock_out,omitempty"`
	CreatedAt     time.Time                `firestore:"created_at"`
	UpdatedAt     time.Time                `firestore:"updated_at"`
}

func storeAttendanceParticipant(v entities.Participant, createdAt, updatedAt time.Time) attendanceParticipantDocument {
	doc := attendanceParticipantDocument{
		SchemaVersion: attendanceParticipantSchemaVersion,
		EmployeeID:    string(v.EmployeeID()), EmployeeCode: v.EmployeeCode(), DisplayName: v.DisplayName(),
		Active: v.Active(), CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
	if id, ok := v.ParticipantID(); ok {
		doc.ParticipantID = string(id)
	}
	if !v.DeactivatedAt().IsZero() {
		at := v.DeactivatedAt()
		doc.DeactivatedAt = &at
	}
	return doc
}

func restoreAttendanceParticipant(d attendanceParticipantDocument) (entities.Participant, error) {
	if d.SchemaVersion != attendanceParticipantSchemaVersion || d.EmployeeID == "" || d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	employeeID, err := valueobjects.NewEmployeeID(d.EmployeeID)
	if err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	participantID := valueobjects.ParticipantID("")
	if d.ParticipantID != "" {
		participantID, err = valueobjects.NewParticipantID(d.ParticipantID)
		if err != nil {
			return entities.Participant{}, ErrRepositoryCorrupt
		}
	}
	deactivatedAt := time.Time{}
	if d.DeactivatedAt != nil {
		deactivatedAt = *d.DeactivatedAt
	}
	value, err := entities.RestoreParticipant(employeeID, d.EmployeeCode, d.DisplayName, d.Active, participantID, deactivatedAt)
	if err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	return value, nil
}

func storeAttendancePunch(v entities.Punch) *attendancePunchDocument {
	location := v.Location()
	return &attendancePunchDocument{
		ID: string(v.ID()), Kind: string(v.Kind()), RecordedAt: v.RecordedAt(),
		Latitude: location.Latitude(), Longitude: location.Longitude(), AccuracyMeters: location.AccuracyMeters(),
	}
}

func restoreAttendancePunch(d *attendancePunchDocument) (*entities.Punch, error) {
	if d == nil {
		return nil, nil
	}
	id, err := valueobjects.NewPunchID(d.ID)
	if err != nil {
		return nil, ErrRepositoryCorrupt
	}
	kind := valueobjects.PunchKind(d.Kind)
	location, err := valueobjects.NewLocationFix(d.Latitude, d.Longitude, d.AccuracyMeters)
	if err != nil {
		return nil, ErrRepositoryCorrupt
	}
	value, err := entities.RestorePunch(id, kind, d.RecordedAt, location)
	if err != nil {
		return nil, ErrRepositoryCorrupt
	}
	return &value, nil
}

func storeAttendanceDay(v aggregates.AttendanceDay, createdAt, updatedAt time.Time) attendanceDayDocument {
	doc := attendanceDayDocument{
		SchemaVersion: attendanceDaySchemaVersion, ID: string(v.ID()), ParticipantID: string(v.ParticipantID()),
		Workday: v.Workday().String(), Revision: int64(v.Revision()), CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
	if punch, ok := v.ClockIn(); ok {
		doc.ClockIn = storeAttendancePunch(punch)
	}
	if punch, ok := v.ClockOut(); ok {
		doc.ClockOut = storeAttendancePunch(punch)
	}
	return doc
}

func restoreAttendanceDay(d attendanceDayDocument) (aggregates.AttendanceDay, error) {
	if d.SchemaVersion != attendanceDaySchemaVersion || d.Revision <= 0 || d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		return aggregates.AttendanceDay{}, ErrRepositoryCorrupt
	}
	id, err := valueobjects.NewAttendanceDayID(d.ID)
	if err != nil {
		return aggregates.AttendanceDay{}, ErrRepositoryCorrupt
	}
	participantID, err := valueobjects.NewParticipantID(d.ParticipantID)
	if err != nil {
		return aggregates.AttendanceDay{}, ErrRepositoryCorrupt
	}
	workday, err := valueobjects.NewWorkday(d.Workday)
	if err != nil {
		return aggregates.AttendanceDay{}, ErrRepositoryCorrupt
	}
	clockIn, err := restoreAttendancePunch(d.ClockIn)
	if err != nil {
		return aggregates.AttendanceDay{}, err
	}
	clockOut, err := restoreAttendancePunch(d.ClockOut)
	if err != nil {
		return aggregates.AttendanceDay{}, err
	}
	value, err := aggregates.RestoreAttendanceDay(id, participantID, workday, clockIn, clockOut, uint64(d.Revision))
	if err != nil {
		return aggregates.AttendanceDay{}, ErrRepositoryCorrupt
	}
	return value, nil
}
