package workspaceautomation

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

var ErrOperatorProviderUnavailable = errors.New("workspace automation provider credentials are not configured for operator CLI")

type OperatorSummary struct {
	ID         string
	Action     string
	ScopeCount int
	Approved   bool
	HasOutput  bool
	Revision   uint64
	Status     string
}

type Operator struct {
	repository         repositories.AutomationRequestRepository
	service            *Service
	providerConfigured bool
}

type unavailableOperatorProvider struct{}

func (unavailableOperatorProvider) Execute(context.Context, Execution) (ProviderResult, error) {
	return ProviderResult{}, ErrOperatorProviderUnavailable
}

func NewOperator(repository repositories.AutomationRequestRepository, provider ProviderGateway) (*Operator, error) {
	if repository == nil {
		return nil, errors.New("automation request repository is required")
	}
	configured := provider != nil
	if provider == nil {
		provider = unavailableOperatorProvider{}
	}
	service, err := NewService(repository, provider)
	if err != nil {
		return nil, err
	}
	return &Operator{repository: repository, service: service, providerConfigured: configured}, nil
}

func (o *Operator) Get(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewAutomationRequestID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.repository.Find(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAutomation(value), nil
}

func (o *Operator) Request(ctx context.Context, rawID, actionName string, scopes []string) (OperatorSummary, error) {
	id, err := valueobjects.NewAutomationRequestID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	action, err := valueobjects.NewAction(actionName)
	if err != nil {
		return OperatorSummary{}, err
	}
	requiredScopes, err := valueobjects.NewRequiredScopes(scopes)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Request(ctx, id, action, requiredScopes)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAutomation(value), nil
}

func (o *Operator) Approve(ctx context.Context, rawID, approvedBy string) (OperatorSummary, error) {
	id, err := valueobjects.NewAutomationRequestID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	approval, err := valueobjects.NewApproval(approvedBy)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Approve(ctx, id, approval)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAutomation(value), nil
}

func (o *Operator) Execute(ctx context.Context, rawID string) (OperatorSummary, error) {
	// This preflight must precede Service.Execute: that service intentionally
	// persists the executing state before the remote side effect.
	if !o.providerConfigured {
		return OperatorSummary{}, ErrOperatorProviderUnavailable
	}
	id, err := valueobjects.NewAutomationRequestID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Execute(ctx, id)
	if err != nil {
		return summarizeAutomation(value), err
	}
	return summarizeAutomation(value), nil
}

func summarizeAutomation(value aggregates.AutomationRequest) OperatorSummary {
	return OperatorSummary{
		ID: value.ID().String(), Action: value.Action().Name(), ScopeCount: len(value.RequiredScopes()),
		Approved: value.Approval().Valid(), HasOutput: value.Output().Valid(),
		Revision: value.Revision(), Status: string(value.Status()),
	}
}
