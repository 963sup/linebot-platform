package aggregates_test

import (
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

func fixture(t *testing.T) (aggregates.AutomationRequest, valueobjects.Approval, valueobjects.Output) {
	t.Helper()
	id, _ := valueobjects.NewAutomationRequestID("request-1")
	action, _ := valueobjects.NewAction("calendar.create")
	scopes, _ := valueobjects.NewRequiredScopes([]string{"calendar.write", "calendar.read"})
	approval, _ := valueobjects.NewApproval("line-user-1")
	output, _ := valueobjects.NewOutput("provider operation completed")
	request, err := aggregates.NewAutomationRequest(id, action, scopes)
	if err != nil {
		t.Fatal(err)
	}
	return request, approval, output
}

func TestAutomationRequestLifecycle(t *testing.T) {
	request, approval, output := fixture(t)
	if request.Revision() != 1 {
		t.Fatalf("initial revision = %d, want 1", request.Revision())
	}
	if request.Status() != valueobjects.StatusRequested {
		t.Fatalf("status = %q", request.Status())
	}
	if err := request.Approve(approval); err != nil {
		t.Fatal(err)
	}
	if request.Revision() != 2 {
		t.Fatalf("approved revision = %d, want 2", request.Revision())
	}
	if err := request.StartExecution(); err != nil {
		t.Fatal(err)
	}
	if request.Revision() != 3 {
		t.Fatalf("executing revision = %d, want 3", request.Revision())
	}
	if err := request.Succeed(output); err != nil {
		t.Fatal(err)
	}
	if request.Revision() != 4 {
		t.Fatalf("terminal revision = %d, want 4", request.Revision())
	}
	if request.Status() != valueobjects.StatusSucceeded || request.Output() != output {
		t.Fatalf("unexpected terminal request: %#v", request)
	}
	if err := request.Fail(output); !errors.Is(err, aggregates.ErrInvalidTransition) {
		t.Fatalf("terminal state reopened: %v", err)
	}
}

func TestExecutionRequiresApproval(t *testing.T) {
	request, _, _ := fixture(t)
	if err := request.StartExecution(); !errors.Is(err, aggregates.ErrApprovalRequired) {
		t.Fatalf("StartExecution error = %v, want approval required", err)
	}
	if request.Status() != valueobjects.StatusRequested {
		t.Fatal("rejected transition changed state")
	}
}

func TestTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name string
		want valueobjects.Status
		run  func(*aggregates.AutomationRequest, valueobjects.Output) error
	}{
		{"succeeded", valueobjects.StatusSucceeded, func(a *aggregates.AutomationRequest, o valueobjects.Output) error { return a.Succeed(o) }},
		{"failed", valueobjects.StatusFailed, func(a *aggregates.AutomationRequest, o valueobjects.Output) error { return a.Fail(o) }},
		{"unknown", valueobjects.StatusUnknown, func(a *aggregates.AutomationRequest, o valueobjects.Output) error { return a.MarkOutcomeUnknown(o) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, approval, output := fixture(t)
			if err := request.Approve(approval); err != nil {
				t.Fatal(err)
			}
			if err := request.StartExecution(); err != nil {
				t.Fatal(err)
			}
			if err := test.run(&request, output); err != nil {
				t.Fatal(err)
			}
			if request.Status() != test.want {
				t.Fatalf("status = %q, want %q", request.Status(), test.want)
			}
			if request.Output() != output || request.Revision() != 4 {
				t.Fatal("terminal transition lost output or revision")
			}
		})
	}
}

func TestInvalidTransitionsDoNotMutate(t *testing.T) {
	request, approval, output := fixture(t)
	if err := request.Approve(approval); err != nil {
		t.Fatal(err)
	}
	if err := request.Approve(approval); !errors.Is(err, aggregates.ErrInvalidTransition) {
		t.Fatalf("second approval: %v", err)
	}
	if err := request.Succeed(output); !errors.Is(err, aggregates.ErrInvalidTransition) {
		t.Fatalf("finish before execution: %v", err)
	}
	if request.Status() != valueobjects.StatusApproved {
		t.Fatal("invalid transition mutated aggregate")
	}
	if request.Revision() != 2 {
		t.Fatalf("invalid transition changed revision to %d", request.Revision())
	}
	if err := request.StartExecution(); err != nil {
		t.Fatal(err)
	}
	if err := request.StartExecution(); !errors.Is(err, aggregates.ErrInvalidTransition) {
		t.Fatalf("second execution: %v", err)
	}
	if request.Status() != valueobjects.StatusExecuting {
		t.Fatal("invalid transition mutated aggregate")
	}
}

func TestRequiredScopesAreDetached(t *testing.T) {
	request, _, _ := fixture(t)
	scopes := request.RequiredScopes()
	scopes[0] = "tampered"
	if request.RequiredScopes()[0] == "tampered" {
		t.Fatal("scope backing data leaked")
	}
}

func TestRestoreAutomationRequestValidatesStateShape(t *testing.T) {
	request, approval, output := fixture(t)
	id, action := request.ID(), request.Action()
	scopes, _ := valueobjects.NewRequiredScopes(request.RequiredScopes())
	tests := []struct {
		name      string
		approval  valueobjects.Approval
		output    valueobjects.Output
		status    valueobjects.Status
		wantError bool
	}{
		{"requested", valueobjects.Approval{}, valueobjects.Output{}, valueobjects.StatusRequested, false},
		{"approved", approval, valueobjects.Output{}, valueobjects.StatusApproved, false},
		{"executing", approval, valueobjects.Output{}, valueobjects.StatusExecuting, false},
		{"succeeded", approval, output, valueobjects.StatusSucceeded, false},
		{"failed", approval, output, valueobjects.StatusFailed, false},
		{"unknown", approval, output, valueobjects.StatusUnknown, false},
		{"requested with approval", approval, valueobjects.Output{}, valueobjects.StatusRequested, true},
		{"approved without approval", valueobjects.Approval{}, valueobjects.Output{}, valueobjects.StatusApproved, true},
		{"executing with output", approval, output, valueobjects.StatusExecuting, true},
		{"terminal without output", approval, valueobjects.Output{}, valueobjects.StatusSucceeded, true},
		{"invalid status", approval, output, valueobjects.Status("invalid"), true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restored, err := aggregates.RestoreAutomationRequest(id, action, scopes, test.approval, test.output, test.status, 7)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %v", err, test.wantError)
			}
			if err == nil && restored.Revision() != 7 {
				t.Fatalf("restored revision = %d, want 7", restored.Revision())
			}
		})
	}
}

func TestZeroValueAggregateRejectsMutation(t *testing.T) {
	var request aggregates.AutomationRequest
	approval, _ := valueobjects.NewApproval("approver")
	output, _ := valueobjects.NewOutput("done")
	if err := request.Approve(approval); !errors.Is(err, aggregates.ErrInvalidAutomationRequest) {
		t.Fatalf("Approve: %v", err)
	}
	if err := request.StartExecution(); !errors.Is(err, aggregates.ErrInvalidAutomationRequest) {
		t.Fatalf("StartExecution: %v", err)
	}
	if err := request.Succeed(output); !errors.Is(err, aggregates.ErrInvalidAutomationRequest) {
		t.Fatalf("Succeed: %v", err)
	}
}
