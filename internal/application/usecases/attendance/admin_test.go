package attendance

import (
	"context"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type adminParticipantsFake struct {
	participant entities.Participant
	saved       bool
}

func (f *adminParticipantsFake) FindByParticipantID(context.Context, valueobjects.ParticipantID) (entities.Participant, error) {
	return f.participant, nil
}
func (f *adminParticipantsFake) FindByEmployeeID(context.Context, valueobjects.EmployeeID) (entities.Participant, error) {
	return f.participant, nil
}
func (f *adminParticipantsFake) SaveParticipant(context.Context, entities.Participant) error {
	f.saved = true
	return nil
}

type tokenFake struct{}

func (tokenFake) Generate() (string, error) { return "one-time-code", nil }

func TestAdminNeedsNoLocationOrKeyringDependency(t *testing.T) {
	now := time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	employeeID, _ := valueobjects.NewEmployeeID("employee")
	participant, _ := entities.NewParticipant(employeeID, "E1", "Employee")
	participants := &adminParticipantsFake{participant: participant}
	enrollment := &enrollmentFake{}
	service, err := NewAdminService(AdminDependencies{Enrollment: enrollment, Participants: participants, Clock: fixedClock{now}}, tokenFake{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.InviteEmployee(context.Background(), InviteEmployeeCommand{EmployeeID: "employee", EmployeeCode: "E1", DisplayName: "Employee", ExpiresAt: now.Add(time.Hour)})
	if err != nil || result.EnrollmentCode != "one-time-code" {
		t.Fatalf("InviteEmployee() = %+v, %v", result, err)
	}
	if err := service.DeactivateEmployee(context.Background(), DeactivateEmployeeCommand{EmployeeID: "employee"}); err != nil || !participants.saved {
		t.Fatalf("DeactivateEmployee() = %v", err)
	}
}
