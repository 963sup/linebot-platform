package workspaceautomation

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

var errWorkspacePort = errors.New("workspace port failure")

type automationRepository struct {
	value     aggregates.AutomationRequest
	findErr   error
	saveErrAt int
	saveCalls int
}

func (r *automationRepository) Find(context.Context, valueobjects.AutomationRequestID) (aggregates.AutomationRequest, error) {
	return r.value, r.findErr
}
func (r *automationRepository) Save(_ context.Context, value aggregates.AutomationRequest) error {
	r.saveCalls++
	if r.saveErrAt == r.saveCalls {
		return errWorkspacePort
	}
	r.value = value
	return nil
}

type workspaceProvider struct {
	result    ProviderResult
	err       error
	calls     int
	execution Execution
}

func (p *workspaceProvider) Execute(_ context.Context, execution Execution) (ProviderResult, error) {
	p.calls++
	p.execution = execution
	return p.result, p.err
}

func TestRequestAndApprovePersistTransitions(t *testing.T) {
	t.Parallel()
	repository := &automationRepository{}
	service, _ := NewService(repository, &workspaceProvider{})
	id, action, scopes, approval := automationValues(t)
	requested, err := service.Request(context.Background(), id, action, scopes)
	if err != nil || requested.Status() != valueobjects.StatusRequested {
		t.Fatalf("Request() = (%v, %v), want requested, nil", requested.Status(), err)
	}
	approved, err := service.Approve(context.Background(), id, approval)
	if err != nil || approved.Status() != valueobjects.StatusApproved {
		t.Fatalf("Approve() = (%v, %v), want approved, nil", approved.Status(), err)
	}
	if repository.saveCalls != 2 {
		t.Fatalf("Save calls = %d, want 2", repository.saveCalls)
	}
}

func TestExecuteMapsKnownProviderOutcomes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		outcome ProviderOutcome
		want    valueobjects.Status
	}{
		{"succeeded", ProviderOutcomeSucceeded, valueobjects.StatusSucceeded},
		{"failed", ProviderOutcomeFailed, valueobjects.StatusFailed},
		{"unknown", ProviderOutcomeUnknown, valueobjects.StatusUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &automationRepository{value: approvedRequest(t)}
			provider := &workspaceProvider{result: ProviderResult{Outcome: test.outcome, Detail: test.name}}
			service, _ := NewService(repository, provider)
			request, err := service.Execute(context.Background(), repository.value.ID())
			if err != nil || request.Status() != test.want || request.Output().Detail() != test.name {
				t.Fatalf("Execute() = (%v, %v), want %v, nil", request.Status(), err, test.want)
			}
			if repository.saveCalls != 2 || provider.calls != 1 {
				t.Fatalf("Save/Execute calls = %d/%d, want 2/1", repository.saveCalls, provider.calls)
			}
			if provider.execution.RequestID != request.ID() || len(provider.execution.RequiredScopes) == 0 {
				t.Fatalf("provider execution = %#v, want aggregate identity and scopes", provider.execution)
			}
		})
	}
}

func TestExecuteMapsProviderErrorAndInvalidResultToUnknown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		provider *workspaceProvider
		wantErr  error
	}{
		{"transport ambiguity", &workspaceProvider{err: errWorkspacePort}, errWorkspacePort},
		{"invalid result", &workspaceProvider{result: ProviderResult{Outcome: 99, Detail: "invalid"}}, ErrInvalidProviderResult},
		{"empty detail", &workspaceProvider{result: ProviderResult{Outcome: ProviderOutcomeSucceeded}}, ErrInvalidProviderResult},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &automationRepository{value: approvedRequest(t)}
			service, _ := NewService(repository, test.provider)
			request, err := service.Execute(context.Background(), repository.value.ID())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			if request.Status() != valueobjects.StatusUnknown || repository.value.Status() != valueobjects.StatusUnknown {
				t.Fatalf("ambiguous result status = %v/%v, want unknown", request.Status(), repository.value.Status())
			}
			if repository.saveCalls != 2 {
				t.Fatalf("Save calls = %d, want 2", repository.saveCalls)
			}
		})
	}
}

func TestExecuteStopsAtPersistenceBoundaries(t *testing.T) {
	t.Parallel()
	id, _, _, _ := automationValues(t)
	provider := &workspaceProvider{result: ProviderResult{Outcome: ProviderOutcomeSucceeded, Detail: "done"}}
	repository := &automationRepository{value: approvedRequest(t), saveErrAt: 1}
	service, _ := NewService(repository, provider)
	if _, err := service.Execute(context.Background(), id); !errors.Is(err, errWorkspacePort) {
		t.Fatalf("Execute(admission save error) = %v, want %v", err, errWorkspacePort)
	}
	if provider.calls != 0 {
		t.Fatal("provider called before executing state was persisted")
	}

	repository = &automationRepository{value: approvedRequest(t), saveErrAt: 2}
	service, _ = NewService(repository, provider)
	if _, err := service.Execute(context.Background(), id); !errors.Is(err, errWorkspacePort) {
		t.Fatalf("Execute(final save error) = %v, want %v", err, errWorkspacePort)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
}

func TestExecuteReconcilesInterruptedExecutionWithoutCallingProviderAgain(t *testing.T) {
	t.Parallel()
	request := approvedRequest(t)
	if err := request.StartExecution(); err != nil {
		t.Fatal(err)
	}
	repository := &automationRepository{value: request}
	provider := &workspaceProvider{result: ProviderResult{Outcome: ProviderOutcomeSucceeded, Detail: "must-not-run"}}
	service, _ := NewService(repository, provider)

	result, err := service.Execute(context.Background(), request.ID())
	if !errors.Is(err, ErrExecutionInterrupted) {
		t.Fatalf("Execute(interrupted) error = %v, want %v", err, ErrExecutionInterrupted)
	}
	if result.Status() != valueobjects.StatusUnknown || repository.value.Status() != valueobjects.StatusUnknown {
		t.Fatalf("interrupted status = %v/%v, want unknown", result.Status(), repository.value.Status())
	}
	if provider.calls != 0 || repository.saveCalls != 1 {
		t.Fatalf("provider/save calls = %d/%d, want 0/1", provider.calls, repository.saveCalls)
	}
}

func TestWorkspaceServiceDependenciesContextAndFindErrors(t *testing.T) {
	t.Parallel()
	provider, repository := &workspaceProvider{}, &automationRepository{}
	if _, err := NewService(nil, provider); err == nil {
		t.Fatal("NewService(nil, provider) error = nil")
	}
	if _, err := NewService(repository, nil); err == nil {
		t.Fatal("NewService(repository, nil) error = nil")
	}
	service, _ := NewService(repository, provider)
	id, action, scopes, approval := automationValues(t)
	if _, err := service.Request(nil, id, action, scopes); !errors.Is(err, ErrContextRequired) {
		t.Fatalf("Request(nil) = %v, want %v", err, ErrContextRequired)
	}
	repository.findErr = errWorkspacePort
	if _, err := service.Approve(context.Background(), id, approval); !errors.Is(err, errWorkspacePort) {
		t.Fatalf("Approve(find error) = %v, want %v", err, errWorkspacePort)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Execute(ctx, id); !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute(canceled) = %v, want %v", err, context.Canceled)
	}
}

func automationValues(t *testing.T) (valueobjects.AutomationRequestID, valueobjects.Action, valueobjects.RequiredScopes, valueobjects.Approval) {
	t.Helper()
	id, _ := valueobjects.NewAutomationRequestID("request-1")
	action, _ := valueobjects.NewAction("calendar.create")
	scopes, _ := valueobjects.NewRequiredScopes([]string{"calendar.write"})
	approval, _ := valueobjects.NewApproval("operator-1")
	return id, action, scopes, approval
}

func approvedRequest(t *testing.T) aggregates.AutomationRequest {
	t.Helper()
	id, action, scopes, approval := automationValues(t)
	request, err := aggregates.NewAutomationRequest(id, action, scopes)
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Approve(approval); err != nil {
		t.Fatal(err)
	}
	return request
}
