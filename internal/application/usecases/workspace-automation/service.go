// Package workspaceautomation orchestrates approved Workspace automation requests.
package workspaceautomation

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

var (
	ErrContextRequired       = errors.New("context is required")
	ErrInvalidProviderResult = errors.New("invalid workspace provider result")
	ErrExecutionInterrupted  = errors.New("execution was interrupted; remote outcome is unknown")
)

type ProviderOutcome uint8

const (
	ProviderOutcomeSucceeded ProviderOutcome = iota + 1
	ProviderOutcomeFailed
	ProviderOutcomeUnknown
)

type Execution struct {
	RequestID      valueobjects.AutomationRequestID
	Action         valueobjects.Action
	RequiredScopes []string
}

type ProviderResult struct {
	Outcome ProviderOutcome
	Detail  string
}

// ProviderGateway executes one already-approved workspace action. Transport
// ambiguity is returned as an error and is persisted by Service as unknown.
type ProviderGateway interface {
	Execute(context.Context, Execution) (ProviderResult, error)
}

type Service struct {
	repository repositories.AutomationRequestRepository
	provider   ProviderGateway
}

func NewService(repository repositories.AutomationRequestRepository, provider ProviderGateway) (*Service, error) {
	if repository == nil || provider == nil {
		return nil, errors.New("automation request repository and provider gateway are required")
	}
	return &Service{repository: repository, provider: provider}, nil
}

func (s *Service) Request(ctx context.Context, id valueobjects.AutomationRequestID, action valueobjects.Action, scopes valueobjects.RequiredScopes) (aggregates.AutomationRequest, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	request, err := aggregates.NewAutomationRequest(id, action, scopes)
	if err != nil {
		return aggregates.AutomationRequest{}, err
	}
	if err := s.repository.Save(ctx, request); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	return request, nil
}

func (s *Service) Approve(ctx context.Context, id valueobjects.AutomationRequestID, approval valueobjects.Approval) (aggregates.AutomationRequest, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	request, err := s.repository.Find(ctx, id)
	if err != nil {
		return aggregates.AutomationRequest{}, err
	}
	if err := request.Approve(approval); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	if err := s.repository.Save(ctx, request); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	return request, nil
}

func (s *Service) Execute(ctx context.Context, id valueobjects.AutomationRequestID) (aggregates.AutomationRequest, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	request, err := s.repository.Find(ctx, id)
	if err != nil {
		return aggregates.AutomationRequest{}, err
	}
	if request.Status() == valueobjects.StatusExecuting {
		output, outputErr := valueobjects.NewOutput(ErrExecutionInterrupted.Error())
		if outputErr != nil {
			return aggregates.AutomationRequest{}, outputErr
		}
		if transitionErr := request.MarkOutcomeUnknown(output); transitionErr != nil {
			return aggregates.AutomationRequest{}, transitionErr
		}
		if saveErr := s.repository.Save(ctx, request); saveErr != nil {
			return aggregates.AutomationRequest{}, saveErr
		}
		return request, ErrExecutionInterrupted
	}
	if err := request.StartExecution(); err != nil {
		return aggregates.AutomationRequest{}, err
	}
	// Persist admission before causing an external side effect.
	if err := s.repository.Save(ctx, request); err != nil {
		return aggregates.AutomationRequest{}, err
	}

	result, providerErr := s.provider.Execute(ctx, Execution{
		RequestID: request.ID(), Action: request.Action(), RequiredScopes: request.RequiredScopes(),
	})
	if providerErr != nil {
		result = ProviderResult{Outcome: ProviderOutcomeUnknown, Detail: providerErr.Error()}
	}
	resultErr := validateProviderResult(result)
	if resultErr != nil {
		result = ProviderResult{Outcome: ProviderOutcomeUnknown, Detail: resultErr.Error()}
	}
	output, err := valueobjects.NewOutput(result.Detail)
	if err != nil {
		return aggregates.AutomationRequest{}, errors.Join(providerErr, resultErr, err)
	}
	switch result.Outcome {
	case ProviderOutcomeSucceeded:
		err = request.Succeed(output)
	case ProviderOutcomeFailed:
		err = request.Fail(output)
	case ProviderOutcomeUnknown:
		err = request.MarkOutcomeUnknown(output)
	}
	if err != nil {
		return aggregates.AutomationRequest{}, errors.Join(providerErr, resultErr, err)
	}
	if saveErr := s.repository.Save(ctx, request); saveErr != nil {
		return aggregates.AutomationRequest{}, errors.Join(providerErr, resultErr, saveErr)
	}
	return request, errors.Join(providerErr, resultErr)
}

func validateProviderResult(result ProviderResult) error {
	if strings.TrimSpace(result.Detail) == "" {
		return ErrInvalidProviderResult
	}
	switch result.Outcome {
	case ProviderOutcomeSucceeded, ProviderOutcomeFailed, ProviderOutcomeUnknown:
		return nil
	default:
		return ErrInvalidProviderResult
	}
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}
