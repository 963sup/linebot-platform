package workspaceautomation

import (
	"context"
	"errors"
	"testing"
)

type operationsGatewayFake struct {
	calendarCalls int
	tasksCalls    int
	formsCalls    int
	command       OperationCommand
	result        OperationResult
	err           error
}

func (f *operationsGatewayFake) ExecuteCalendar(_ context.Context, command OperationCommand) (OperationResult, error) {
	f.calendarCalls++
	f.command = command
	return f.result, f.err
}

func (f *operationsGatewayFake) ExecuteTasks(_ context.Context, command OperationCommand) (OperationResult, error) {
	f.tasksCalls++
	f.command = command
	return f.result, f.err
}

func (f *operationsGatewayFake) ExecuteForms(_ context.Context, command OperationCommand) (OperationResult, error) {
	f.formsCalls++
	f.command = command
	return f.result, f.err
}

func TestSupportedOperationsPreserveAuditedSurface(t *testing.T) {
	t.Parallel()
	if got := len(SupportedOperations()); got != 62 {
		t.Fatalf("SupportedOperations() count = %d, want 62", got)
	}
	for _, test := range []struct {
		api       API
		operation Operation
		mode      OperationMode
		dangerous bool
	}{
		{APICalendar, "events.list", ModeRead, false},
		{APICalendar, "events.delete", ModeWrite, true},
		{APITasks, "tasks.insert", ModeWrite, false},
		{APIForms, "forms.batchUpdate", ModeWrite, true},
	} {
		policy, ok := LookupOperation(test.api, test.operation)
		if !ok || policy.Mode != test.mode || policy.Dangerous != test.dangerous {
			t.Fatalf("LookupOperation(%q, %q) = %#v, %v", test.api, test.operation, policy, ok)
		}
	}
}

func TestOperationsRoutesTypedCommands(t *testing.T) {
	t.Parallel()
	gateway := &operationsGatewayFake{result: OperationResult{Payload: []byte(`{"id":"event-1"}`)}}
	operations, err := NewOperations(gateway, gateway, gateway)
	if err != nil {
		t.Fatal(err)
	}
	command := OperationCommand{
		Operation:  "events.insert",
		Parameters: []Parameter{{Name: "calendarId", Value: "primary"}},
		Options:    []Option{{Name: "sendUpdates", Values: []string{"all"}}},
		Payload:    []byte(`{"summary":"review"}`),
	}
	result, err := operations.Execute(context.Background(), APICalendar, command, OperationApproval{Apply: true, Confirm: command.Operation})
	if err != nil {
		t.Fatal(err)
	}
	if gateway.calendarCalls != 1 || gateway.tasksCalls != 0 || gateway.formsCalls != 0 {
		t.Fatalf("gateway calls = %d/%d/%d", gateway.calendarCalls, gateway.tasksCalls, gateway.formsCalls)
	}
	if string(result.Payload) == "" || gateway.command.Parameters[0].Value != "primary" {
		t.Fatalf("result/command = %q/%#v", result.Payload, gateway.command)
	}
	command.Payload[0] = 'X'
	command.Options[0].Values[0] = "none"
	if gateway.command.Payload[0] == 'X' || gateway.command.Options[0].Values[0] == "none" {
		t.Fatal("gateway command aliases caller-owned data")
	}
}

func TestOperationsEnforcesReadWriteAndDangerousApproval(t *testing.T) {
	t.Parallel()
	gateway := &operationsGatewayFake{}
	operations, _ := NewOperations(gateway, gateway, gateway)
	tests := []struct {
		name     string
		api      API
		command  OperationCommand
		approval OperationApproval
	}{
		{"read without approval", APICalendar, OperationCommand{Operation: "events.list"}, OperationApproval{}},
		{"read with write approval", APICalendar, OperationCommand{Operation: "events.list"}, OperationApproval{Read: true, Apply: true}},
		{"write without matching confirmation", APITasks, OperationCommand{Operation: "tasks.insert"}, OperationApproval{Apply: true}},
		{"dangerous without second confirmation", APIForms, OperationCommand{Operation: "forms.batchUpdate"}, OperationApproval{Apply: true, Confirm: "forms.batchUpdate"}},
		{"unsupported", APIForms, OperationCommand{Operation: "forms.delete"}, OperationApproval{Apply: true, Confirm: "forms.delete"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := operations.Execute(context.Background(), test.api, test.command, test.approval); err == nil {
				t.Fatal("Execute() error = nil")
			}
		})
	}
	if gateway.calendarCalls+gateway.tasksCalls+gateway.formsCalls != 0 {
		t.Fatal("gateway called for rejected operation")
	}

	command := OperationCommand{Operation: "forms.batchUpdate"}
	approval := OperationApproval{Apply: true, Confirm: command.Operation, ConfirmDangerous: command.Operation}
	if _, err := operations.Execute(context.Background(), APIForms, command, approval); err != nil {
		t.Fatalf("dangerous Execute() = %v", err)
	}
}

func TestOperationsRejectsInvalidDependenciesContextAndInput(t *testing.T) {
	t.Parallel()
	gateway := &operationsGatewayFake{err: errors.New("provider failed")}
	if _, err := NewOperations(nil, gateway, gateway); err == nil {
		t.Fatal("nil gateway accepted")
	}
	operations, _ := NewOperations(gateway, gateway, gateway)
	if _, err := operations.Execute(nil, APICalendar, OperationCommand{Operation: "events.list"}, OperationApproval{Read: true}); !errors.Is(err, ErrContextRequired) {
		t.Fatalf("Execute(nil) = %v, want %v", err, ErrContextRequired)
	}
	bad := OperationCommand{Operation: "events.list", Parameters: []Parameter{{Name: "calendarId"}, {Name: "calendarId"}}}
	if _, err := operations.Execute(context.Background(), APICalendar, bad, OperationApproval{Read: true}); err == nil {
		t.Fatal("duplicate parameter accepted")
	}
	if _, err := operations.Execute(context.Background(), APICalendar, OperationCommand{Operation: "events.list"}, OperationApproval{Read: true}); !errors.Is(err, gateway.err) {
		t.Fatalf("provider error = %v, want %v", err, gateway.err)
	}
}
