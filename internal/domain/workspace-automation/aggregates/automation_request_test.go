package aggregates_test

import (
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/events"
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
	if request.Status() != valueobjects.StatusRequested {
		t.Fatalf("status = %q", request.Status())
	}
	if err := request.Approve(approval); err != nil {
		t.Fatal(err)
	}
	if err := request.StartExecution(); err != nil {
		t.Fatal(err)
	}
	if err := request.Succeed(output); err != nil {
		t.Fatal(err)
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
		name  string
		want  valueobjects.Status
		run   func(*aggregates.AutomationRequest, valueobjects.Output) error
		event func(events.AutomationEvent) bool
	}{
		{"succeeded", valueobjects.StatusSucceeded, func(a *aggregates.AutomationRequest, o valueobjects.Output) error { return a.Succeed(o) }, func(e events.AutomationEvent) bool { _, ok := e.(events.AutomationSucceeded); return ok }},
		{"failed", valueobjects.StatusFailed, func(a *aggregates.AutomationRequest, o valueobjects.Output) error { return a.Fail(o) }, func(e events.AutomationEvent) bool { _, ok := e.(events.AutomationFailed); return ok }},
		{"unknown", valueobjects.StatusUnknown, func(a *aggregates.AutomationRequest, o valueobjects.Output) error { return a.MarkOutcomeUnknown(o) }, func(e events.AutomationEvent) bool { _, ok := e.(events.AutomationOutcomeUnknown); return ok }},
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
			pending := request.PendingEvents()
			if len(pending) != 4 || !test.event(pending[3]) {
				t.Fatalf("unexpected events: %#v", pending)
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
	if request.Status() != valueobjects.StatusApproved || len(request.PendingEvents()) != 2 {
		t.Fatal("invalid transition mutated aggregate")
	}
	if err := request.StartExecution(); err != nil {
		t.Fatal(err)
	}
	if err := request.StartExecution(); !errors.Is(err, aggregates.ErrInvalidTransition) {
		t.Fatalf("second execution: %v", err)
	}
	if request.Status() != valueobjects.StatusExecuting || len(request.PendingEvents()) != 3 {
		t.Fatal("invalid transition mutated aggregate")
	}
}

func TestEventsAreOrderedAndDetached(t *testing.T) {
	request, approval, _ := fixture(t)
	if err := request.Approve(approval); err != nil {
		t.Fatal(err)
	}
	if err := request.StartExecution(); err != nil {
		t.Fatal(err)
	}
	pending := request.PendingEvents()
	if len(pending) != 3 {
		t.Fatalf("events = %d", len(pending))
	}
	if _, ok := pending[0].(events.AutomationRequested); !ok {
		t.Fatal("first event is not AutomationRequested")
	}
	if _, ok := pending[1].(events.AutomationApproved); !ok {
		t.Fatal("second event is not AutomationApproved")
	}
	if _, ok := pending[2].(events.AutomationExecutionStarted); !ok {
		t.Fatal("third event is not AutomationExecutionStarted")
	}
	pending[0] = nil
	if request.PendingEvents()[0] == nil {
		t.Fatal("event backing data leaked")
	}
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
			restored, err := aggregates.RestoreAutomationRequest(id, action, scopes, test.approval, test.output, test.status)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %v", err, test.wantError)
			}
			if err == nil && len(restored.PendingEvents()) != 0 {
				t.Fatal("restore emitted events")
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
