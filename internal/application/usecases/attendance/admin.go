package attendance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type TokenGenerator interface{ Generate() (string, error) }
type InviteEmployeeCommand struct {
	EmployeeID, EmployeeCode, DisplayName string
	ExpiresAt                             time.Time
}
type InviteEmployeeResult struct {
	EnrollmentCode string
	ExpiresAt      time.Time
}
type DeactivateEmployeeCommand struct{ EmployeeID string }

type AdminService struct {
	deps   AdminDependencies
	tokens TokenGenerator
}

func NewAdminService(deps AdminDependencies, tokens TokenGenerator) (*AdminService, error) {
	if deps.Enrollment == nil || deps.Participants == nil || deps.Clock == nil || tokens == nil {
		return nil, errors.New("attendance admin dependencies are required")
	}
	return &AdminService{deps: deps, tokens: tokens}, nil
}

func (s *AdminService) InviteEmployee(ctx context.Context, c InviteEmployeeCommand) (InviteEmployeeResult, error) {
	if err := validContext(ctx); err != nil {
		return InviteEmployeeResult{}, err
	}
	now := s.deps.Clock.Now().UTC()
	id, err := valueobjects.NewEmployeeID(c.EmployeeID)
	if err != nil || strings.TrimSpace(c.EmployeeCode) == "" || strings.TrimSpace(c.DisplayName) == "" || !c.ExpiresAt.After(now) || c.ExpiresAt.After(now.Add(24*time.Hour)) {
		return InviteEmployeeResult{}, ErrInvalidInput
	}
	participant, err := entities.NewParticipant(id, c.EmployeeCode, c.DisplayName)
	if err != nil {
		return InviteEmployeeResult{}, ErrInvalidInput
	}
	token, err := s.tokens.Generate()
	if err != nil || strings.TrimSpace(token) == "" {
		return InviteEmployeeResult{}, ErrEnrollmentUnavailable
	}
	invitation, err := entities.NewEnrollmentInvitation(id, enrollmentCodeHash(token), c.ExpiresAt)
	if err != nil {
		return InviteEmployeeResult{}, ErrInvalidInput
	}
	if err := s.deps.Enrollment.Invite(ctx, participant, invitation); err != nil {
		return InviteEmployeeResult{}, repositoryError(ctx, err, ErrInvalidInput)
	}
	return InviteEmployeeResult{EnrollmentCode: token, ExpiresAt: c.ExpiresAt.UTC()}, nil
}

func (s *AdminService) DeactivateEmployee(ctx context.Context, c DeactivateEmployeeCommand) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	id, err := valueobjects.NewEmployeeID(c.EmployeeID)
	if err != nil {
		return ErrInvalidInput
	}
	participant, err := s.deps.Participants.FindByEmployeeID(ctx, id)
	if err != nil {
		return repositoryError(ctx, err, ErrInvalidInput)
	}
	if err := participant.Deactivate(s.deps.Clock.Now().UTC()); err != nil {
		return err
	}
	return repositoryError(ctx, s.deps.Participants.SaveParticipant(ctx, participant), ErrInvalidInput)
}

func enrollmentCodeHash(code string) valueobjects.EnrollmentCodeHash {
	sum := sha256.Sum256([]byte("attendance-enrollment-v1\x00" + code))
	value, _ := valueobjects.NewEnrollmentCodeHash("v1:" + hex.EncodeToString(sum[:]))
	return value
}

func validContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidInput
	}
	return ctx.Err()
}
