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
	ExpiresAt     *time.Time `firestore:"expires_at,omitempty"`
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
	SchemaVersion int        `firestore:"schema_version"`
	ParticipantID string     `firestore:"participant_id"`
	EmployeeID    string     `firestore:"employee_id"`
	CreatedAt     time.Time  `firestore:"created_at"`
	ExpiresAt     *time.Time `firestore:"expires_at,omitempty"`
}

type attendanceWorkplaceDocument struct {
	SchemaVersion     int       `firestore:"schema_version"`
	ID                string    `firestore:"id"`
	DisplayName       string    `firestore:"display_name"`
	Active            bool      `firestore:"active"`
	Latitude          float64   `firestore:"latitude"`
	Longitude         float64   `firestore:"longitude"`
	RadiusMeters      float64   `firestore:"radius_meters"`
	MaxAccuracyMeters float64   `firestore:"max_accuracy_meters"`
	PolicyVersion     string    `firestore:"policy_version"`
	CreatedAt         time.Time `firestore:"created_at"`
	UpdatedAt         time.Time `firestore:"updated_at"`
}

type attendanceShiftDocument struct {
	SchemaVersion      int       `firestore:"schema_version"`
	ID                 string    `firestore:"id"`
	WorkplaceID        string    `firestore:"workplace_id"`
	Label              string    `firestore:"label"`
	Active             bool      `firestore:"active"`
	StartsAt           time.Time `firestore:"starts_at"`
	EndsAt             time.Time `firestore:"ends_at"`
	ClockInOpensAt     time.Time `firestore:"clock_in_opens_at"`
	ClockInClosesAt    time.Time `firestore:"clock_in_closes_at"`
	ClockOutOpensAt    time.Time `firestore:"clock_out_opens_at"`
	ClockOutClosesAt   time.Time `firestore:"clock_out_closes_at"`
	SubmissionOpensAt  time.Time `firestore:"submission_opens_at"`
	SubmissionClosesAt time.Time `firestore:"submission_closes_at"`
	PolicyVersion      string    `firestore:"policy_version"`
	CreatedAt          time.Time `firestore:"created_at"`
	UpdatedAt          time.Time `firestore:"updated_at"`
}

type attendanceAssignmentDocument struct {
	SchemaVersion      int       `firestore:"schema_version"`
	ID                 string    `firestore:"id"`
	EmployeeID         string    `firestore:"employee_id"`
	ShiftInstanceID    string    `firestore:"shift_instance_id"`
	Active             bool      `firestore:"active"`
	SubmissionOpensAt  time.Time `firestore:"submission_opens_at"`
	SubmissionClosesAt time.Time `firestore:"submission_closes_at"`
	CreatedAt          time.Time `firestore:"created_at"`
	UpdatedAt          time.Time `firestore:"updated_at"`
}

type attendanceAssignmentGuardDocument struct {
	SchemaVersion   int       `firestore:"schema_version"`
	AssignmentID    string    `firestore:"assignment_id"`
	EmployeeID      string    `firestore:"employee_id"`
	ShiftInstanceID string    `firestore:"shift_instance_id"`
	CreatedAt       time.Time `firestore:"created_at"`
}

type attendanceLocationDocument struct {
	DistanceMeters float64 `firestore:"distance_meters"`
	AccuracyMeters float64 `firestore:"accuracy_meters"`
	Outside        bool    `firestore:"outside_workplace"`
	LowAccuracy    bool    `firestore:"low_accuracy"`
	EvidenceRef    string  `firestore:"evidence_ref"`
	PolicyVersion  string  `firestore:"policy_version"`
}

type attendanceReviewDocument struct {
	State      string                           `firestore:"state"`
	Provenance string                           `firestore:"provenance"`
	At         time.Time                        `firestore:"at"`
	Reviewer   string                           `firestore:"reviewer"`
	Reason     string                           `firestore:"reason"`
	Before     attendanceReviewSnapshotDocument `firestore:"before"`
	After      attendanceReviewSnapshotDocument `firestore:"after"`
}

type attendanceReviewSnapshotDocument struct {
	EffectiveAt time.Time                  `firestore:"effective_at"`
	Location    attendanceLocationDocument `firestore:"location"`
	TimeAnomaly bool                       `firestore:"time_anomaly"`
}

type attendancePunchDocument struct {
	ID           string                     `firestore:"id"`
	Kind         string                     `firestore:"kind"`
	SubmissionID string                     `firestore:"submission_id"`
	RecordedAt   time.Time                  `firestore:"recorded_at"`
	EffectiveAt  time.Time                  `firestore:"effective_at"`
	Location     attendanceLocationDocument `firestore:"location"`
	TimeAnomaly  bool                       `firestore:"time_anomaly"`
	ReviewState  string                     `firestore:"review_state"`
	History      []attendanceReviewDocument `firestore:"history"`
}

type attendanceRecordDocument struct {
	SchemaVersion  int                           `firestore:"schema_version"`
	Participant    attendanceParticipantDocument `firestore:"participant"`
	Workplace      attendanceWorkplaceDocument   `firestore:"workplace"`
	Shift          attendanceShiftDocument       `firestore:"shift"`
	Assignment     attendanceAssignmentDocument  `firestore:"assignment"`
	ClockIn        *attendancePunchDocument      `firestore:"clock_in,omitempty"`
	ClockOut       *attendancePunchDocument      `firestore:"clock_out,omitempty"`
	Revision       int64                         `firestore:"revision"`
	ReviewRequired bool                          `firestore:"review_required"`
	CreatedAt      time.Time                     `firestore:"created_at"`
	UpdatedAt      time.Time                     `firestore:"updated_at"`
	ExpiresAt      time.Time                     `firestore:"expires_at"`
}

type attendanceEvidenceDocument struct {
	SchemaVersion int       `firestore:"schema_version"`
	PunchID       string    `firestore:"punch_id"`
	KeyID         string    `firestore:"key_id"`
	Ciphertext    []byte    `firestore:"ciphertext"`
	CreatedAt     time.Time `firestore:"created_at"`
	ExpiresAt     time.Time `firestore:"expires_at"`
}

type attendanceEvidenceAccessDocument struct {
	SchemaVersion int       `firestore:"schema_version"`
	EvidenceRef   string    `firestore:"evidence_ref"`
	Reviewer      string    `firestore:"reviewer"`
	Reason        string    `firestore:"reason"`
	AccessedAt    time.Time `firestore:"accessed_at"`
	ExpiresAt     time.Time `firestore:"expires_at"`
}

type attendanceReviewActionDocument struct {
	SchemaVersion int                              `firestore:"schema_version"`
	AttendanceID  string                           `firestore:"attendance_id"`
	PunchID       string                           `firestore:"punch_id"`
	Sequence      int                              `firestore:"sequence"`
	State         string                           `firestore:"state"`
	Provenance    string                           `firestore:"provenance"`
	At            time.Time                        `firestore:"at"`
	Reviewer      string                           `firestore:"reviewer"`
	Reason        string                           `firestore:"reason"`
	Before        attendanceReviewSnapshotDocument `firestore:"before"`
	After         attendanceReviewSnapshotDocument `firestore:"after"`
	ExpiresAt     time.Time                        `firestore:"expires_at"`
}

func storeAttendanceParticipant(v entities.Participant, createdAt, updatedAt time.Time) attendanceParticipantDocument {
	doc := attendanceParticipantDocument{SchemaVersion: attendanceSchemaVersion, EmployeeID: string(v.EmployeeID()), EmployeeCode: v.EmployeeCode(), DisplayName: v.DisplayName(), Active: v.Active(), CreatedAt: createdAt, UpdatedAt: updatedAt}
	if id, ok := v.ParticipantID(); ok {
		doc.ParticipantID = string(id)
	}
	if !v.DeactivatedAt().IsZero() {
		at := v.DeactivatedAt()
		expires := attendanceExpiry(at)
		doc.DeactivatedAt = &at
		doc.ExpiresAt = &expires
	}
	return doc
}
func restoreAttendanceParticipant(d attendanceParticipantDocument) (entities.Participant, error) {
	if d.SchemaVersion != attendanceSchemaVersion || d.EmployeeID == "" || d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	eid, err := valueobjects.NewEmployeeID(d.EmployeeID)
	if err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	pid := valueobjects.ParticipantID("")
	if d.ParticipantID != "" {
		pid, err = valueobjects.NewParticipantID(d.ParticipantID)
		if err != nil {
			return entities.Participant{}, ErrRepositoryCorrupt
		}
	}
	deactivated := time.Time{}
	if d.DeactivatedAt != nil {
		deactivated = *d.DeactivatedAt
	}
	value, err := entities.RestoreParticipant(eid, d.EmployeeCode, d.DisplayName, d.Active, pid, deactivated)
	if err != nil {
		return entities.Participant{}, ErrRepositoryCorrupt
	}
	return value, nil
}
func storeAttendanceWorkplace(v entities.Workplace, created, updated time.Time) attendanceWorkplaceDocument {
	return attendanceWorkplaceDocument{SchemaVersion: attendanceSchemaVersion, ID: string(v.ID()), DisplayName: v.DisplayName(), Active: v.Active(), Latitude: v.Center().Latitude(), Longitude: v.Center().Longitude(), RadiusMeters: v.RadiusMeters(), MaxAccuracyMeters: v.MaxAccuracyMeters(), PolicyVersion: v.PolicyVersion(), CreatedAt: created, UpdatedAt: updated}
}
func restoreAttendanceWorkplace(d attendanceWorkplaceDocument) (entities.Workplace, error) {
	if d.SchemaVersion != attendanceSchemaVersion {
		return entities.Workplace{}, ErrRepositoryCorrupt
	}
	id, e := valueobjects.NewWorkplaceID(d.ID)
	if e != nil {
		return entities.Workplace{}, ErrRepositoryCorrupt
	}
	point, e := valueobjects.NewGeoPoint(d.Latitude, d.Longitude)
	if e != nil {
		return entities.Workplace{}, ErrRepositoryCorrupt
	}
	v, e := entities.DefineWorkplace(id, d.DisplayName, point, d.RadiusMeters, d.MaxAccuracyMeters, d.PolicyVersion)
	if e != nil {
		return entities.Workplace{}, ErrRepositoryCorrupt
	}
	if !d.Active {
		if e := v.Deactivate(); e != nil {
			return entities.Workplace{}, ErrRepositoryCorrupt
		}
	}
	return v, nil
}
func storeAttendanceShift(v entities.ShiftInstance, created, updated time.Time) attendanceShiftDocument {
	return attendanceShiftDocument{SchemaVersion: attendanceSchemaVersion, ID: string(v.ID()), WorkplaceID: string(v.WorkplaceID()), Label: v.Label(), Active: v.Active(), StartsAt: v.StartsAt(), EndsAt: v.EndsAt(), ClockInOpensAt: v.ClockInWindow().OpensAt(), ClockInClosesAt: v.ClockInWindow().ClosesAt(), ClockOutOpensAt: v.ClockOutWindow().OpensAt(), ClockOutClosesAt: v.ClockOutWindow().ClosesAt(), SubmissionOpensAt: v.SubmissionOpensAt(), SubmissionClosesAt: v.SubmissionClosesAt(), PolicyVersion: v.PolicyVersion(), CreatedAt: created, UpdatedAt: updated}
}

func restoreAttendanceShift(d attendanceShiftDocument) (entities.ShiftInstance, error) {
	if d.SchemaVersion != attendanceSchemaVersion {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	id, e := valueobjects.NewShiftInstanceID(d.ID)
	if e != nil {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	wid, e := valueobjects.NewWorkplaceID(d.WorkplaceID)
	if e != nil {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	clockInWindow, e := valueobjects.NewTimeWindow(d.ClockInOpensAt, d.ClockInClosesAt)
	if e != nil {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	clockOutWindow, e := valueobjects.NewTimeWindow(d.ClockOutOpensAt, d.ClockOutClosesAt)
	if e != nil {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	v, e := entities.DefineShiftInstance(id, wid, d.Label, clockInWindow, clockOutWindow, d.SubmissionOpensAt, d.SubmissionClosesAt, d.PolicyVersion)
	if e != nil {
		return entities.ShiftInstance{}, ErrRepositoryCorrupt
	}
	if !d.Active {
		if e := v.Deactivate(); e != nil {
			return entities.ShiftInstance{}, ErrRepositoryCorrupt
		}
	}
	return v, nil
}
func storeAttendanceAssignment(v entities.ShiftAssignment, shift entities.ShiftInstance, created, updated time.Time) attendanceAssignmentDocument {
	return attendanceAssignmentDocument{
		SchemaVersion:      attendanceSchemaVersion,
		ID:                 string(v.ID()),
		EmployeeID:         string(v.EmployeeID()),
		ShiftInstanceID:    string(v.ShiftInstanceID()),
		Active:             true,
		SubmissionOpensAt:  shift.SubmissionOpensAt(),
		SubmissionClosesAt: shift.SubmissionClosesAt(),
		CreatedAt:          created,
		UpdatedAt:          updated,
	}
}
func restoreAttendanceAssignment(d attendanceAssignmentDocument) (entities.ShiftAssignment, error) {
	if d.SchemaVersion != attendanceSchemaVersion || !d.Active || d.SubmissionOpensAt.IsZero() || !d.SubmissionClosesAt.After(d.SubmissionOpensAt) {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	id, e := valueobjects.NewShiftAssignmentID(d.ID)
	if e != nil {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	eid, e := valueobjects.NewEmployeeID(d.EmployeeID)
	if e != nil {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	sid, e := valueobjects.NewShiftInstanceID(d.ShiftInstanceID)
	if e != nil {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	v, e := entities.NewShiftAssignment(id, eid, sid)
	if e != nil {
		return entities.ShiftAssignment{}, ErrRepositoryCorrupt
	}
	return v, nil
}
func storeAttendanceLocation(v valueobjects.LocationAssessment) attendanceLocationDocument {
	return attendanceLocationDocument{v.DistanceMeters(), v.AccuracyMeters(), v.OutsideWorkplace(), v.LowAccuracy(), string(v.EvidenceRef()), v.PolicyVersion()}
}
func restoreAttendanceLocation(d attendanceLocationDocument) (valueobjects.LocationAssessment, error) {
	ref, e := valueobjects.NewEvidenceRef(d.EvidenceRef)
	if e != nil {
		return valueobjects.LocationAssessment{}, ErrRepositoryCorrupt
	}
	v, e := valueobjects.NewLocationAssessment(d.DistanceMeters, d.AccuracyMeters, d.Outside, d.LowAccuracy, ref, d.PolicyVersion)
	if e != nil {
		return valueobjects.LocationAssessment{}, ErrRepositoryCorrupt
	}
	return v, nil
}
func storeAttendanceReviewSnapshot(v entities.ReviewSnapshot) attendanceReviewSnapshotDocument {
	return attendanceReviewSnapshotDocument{
		EffectiveAt: v.EffectiveAt(),
		Location:    storeAttendanceLocation(v.Location()),
		TimeAnomaly: v.TimeAnomaly(),
	}
}
func restoreAttendanceReviewSnapshot(d attendanceReviewSnapshotDocument) (entities.ReviewSnapshot, error) {
	location, err := restoreAttendanceLocation(d.Location)
	if err != nil {
		return entities.ReviewSnapshot{}, ErrRepositoryCorrupt
	}
	value, err := entities.NewReviewSnapshot(d.EffectiveAt, location, d.TimeAnomaly)
	if err != nil {
		return entities.ReviewSnapshot{}, ErrRepositoryCorrupt
	}
	return value, nil
}
func storeAttendancePunch(v entities.Punch) *attendancePunchDocument {
	history := v.ReviewHistory()
	storedHistory := make([]attendanceReviewDocument, len(history))
	for i, a := range history {
		storedHistory[i] = attendanceReviewDocument{
			State: string(a.State()), Provenance: string(a.Provenance()), At: a.At(), Reviewer: a.Reviewer(), Reason: a.Reason(),
			Before: storeAttendanceReviewSnapshot(a.Before()), After: storeAttendanceReviewSnapshot(a.After()),
		}
	}
	return &attendancePunchDocument{string(v.ID()), string(v.Kind()), string(v.SubmissionID()), v.RecordedAt(), v.EffectiveAt(), storeAttendanceLocation(v.Location()), v.TimeAnomaly(), string(v.ReviewState()), storedHistory}
}
func restoreAttendancePunch(d *attendancePunchDocument) (*entities.Punch, error) {
	if d == nil {
		return nil, nil
	}
	id, e := valueobjects.NewPunchID(d.ID)
	if e != nil {
		return nil, ErrRepositoryCorrupt
	}
	submission, e := valueobjects.NewSubmissionID(d.SubmissionID)
	if e != nil {
		return nil, ErrRepositoryCorrupt
	}
	location, e := restoreAttendanceLocation(d.Location)
	if e != nil {
		return nil, e
	}
	history := make([]entities.ReviewAction, len(d.History))
	for i, a := range d.History {
		before, e := restoreAttendanceReviewSnapshot(a.Before)
		if e != nil {
			return nil, ErrRepositoryCorrupt
		}
		after, e := restoreAttendanceReviewSnapshot(a.After)
		if e != nil {
			return nil, ErrRepositoryCorrupt
		}
		action, e := entities.NewReviewAction(valueobjects.ReviewState(a.State), valueobjects.ReviewProvenance(a.Provenance), a.At, a.Reviewer, a.Reason, before, after)
		if e != nil {
			return nil, ErrRepositoryCorrupt
		}
		history[i] = action
	}
	p, e := entities.RestorePunch(id, valueobjects.PunchKind(d.Kind), submission, d.RecordedAt, d.EffectiveAt, location, d.TimeAnomaly, valueobjects.ReviewState(d.ReviewState), history)
	if e != nil {
		return nil, ErrRepositoryCorrupt
	}
	return &p, nil
}
func storeAttendanceRecord(v aggregates.Attendance, now time.Time) (attendanceRecordDocument, error) {
	participant := storeAttendanceParticipant(v.Participant(), now, now)
	workplace := storeAttendanceWorkplace(v.Workplace(), now, now)
	shift := storeAttendanceShift(v.Shift(), now, now)
	assignment := storeAttendanceAssignment(v.Assignment(), v.Shift(), now, now)
	var in, out *attendancePunchDocument
	if p, ok := v.ClockIn(); ok {
		in = storeAttendancePunch(p)
	}
	if p, ok := v.ClockOut(); ok {
		out = storeAttendancePunch(p)
	}
	return attendanceRecordDocument{SchemaVersion: attendanceSchemaVersion, Participant: participant, Workplace: workplace, Shift: shift, Assignment: assignment, ClockIn: in, ClockOut: out, Revision: int64(v.Revision()), ReviewRequired: punchRequiresReview(in) || punchRequiresReview(out), UpdatedAt: now}, nil
}
func restoreAttendanceRecord(d attendanceRecordDocument) (aggregates.Attendance, error) {
	if d.SchemaVersion != attendanceSchemaVersion || d.Revision <= 0 || d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() || d.ExpiresAt.IsZero() {
		return aggregates.Attendance{}, ErrRepositoryCorrupt
	}
	participant, e := restoreAttendanceParticipant(d.Participant)
	if e != nil {
		return aggregates.Attendance{}, e
	}
	workplace, e := restoreAttendanceWorkplace(d.Workplace)
	if e != nil {
		return aggregates.Attendance{}, e
	}
	shift, e := restoreAttendanceShift(d.Shift)
	if e != nil {
		return aggregates.Attendance{}, e
	}
	assignment, e := restoreAttendanceAssignment(d.Assignment)
	if e != nil {
		return aggregates.Attendance{}, e
	}
	if d.Assignment.SubmissionOpensAt != shift.SubmissionOpensAt() || d.Assignment.SubmissionClosesAt != shift.SubmissionClosesAt() {
		return aggregates.Attendance{}, ErrRepositoryCorrupt
	}
	in, e := restoreAttendancePunch(d.ClockIn)
	if e != nil {
		return aggregates.Attendance{}, e
	}
	out, e := restoreAttendancePunch(d.ClockOut)
	if e != nil {
		return aggregates.Attendance{}, e
	}
	v, e := aggregates.RestoreAttendance(participant, workplace, shift, assignment, in, out, uint64(d.Revision))
	if e != nil {
		return aggregates.Attendance{}, ErrRepositoryCorrupt
	}
	return v, nil
}
