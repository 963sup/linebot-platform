package workspaceautomation

import (
	"context"
	"errors"
	"strings"
)

type API string

const (
	APICalendar API = "calendar"
	APITasks    API = "tasks"
	APIForms    API = "forms"
)

type Operation string
type OperationMode string

const (
	ModeRead  OperationMode = "read"
	ModeWrite OperationMode = "write"
)

// OperationPolicy is pinned to the audited Workspace API surface.
type OperationPolicy struct {
	API       API
	Operation Operation
	Mode      OperationMode
	Dangerous bool
}

// Parameter and Option avoid untyped provider payloads in the application layer.
// Adapters map these values to the concrete Google SDK request.
type Parameter struct {
	Name  string
	Value string
}

type Option struct {
	Name   string
	Values []string
}

type OperationCommand struct {
	Operation  Operation
	Parameters []Parameter
	Options    []Option
	Payload    []byte
}

type OperationResult struct {
	Payload []byte
}

type CalendarGateway interface {
	ExecuteCalendar(context.Context, OperationCommand) (OperationResult, error)
}

type TasksGateway interface {
	ExecuteTasks(context.Context, OperationCommand) (OperationResult, error)
}

type FormsGateway interface {
	ExecuteForms(context.Context, OperationCommand) (OperationResult, error)
}

type OperationApproval struct {
	Read             bool
	Apply            bool
	Confirm          Operation
	ConfirmDangerous Operation
}

type Operations struct {
	calendar CalendarGateway
	tasks    TasksGateway
	forms    FormsGateway
}

func NewOperations(calendar CalendarGateway, tasks TasksGateway, forms FormsGateway) (*Operations, error) {
	if calendar == nil || tasks == nil || forms == nil {
		return nil, errors.New("Calendar, Tasks and Forms gateways are required")
	}
	return &Operations{calendar: calendar, tasks: tasks, forms: forms}, nil
}

func SupportedOperations() []OperationPolicy {
	return append([]OperationPolicy(nil), operationPolicies...)
}

func LookupOperation(api API, operation Operation) (OperationPolicy, bool) {
	for _, policy := range operationPolicies {
		if policy.API == api && policy.Operation == operation {
			return policy, true
		}
	}
	return OperationPolicy{}, false
}

func ValidateOperation(api API, command OperationCommand, approval OperationApproval) error {
	policy, ok := LookupOperation(api, command.Operation)
	if !ok {
		return errors.New("unsupported Google Workspace operation")
	}
	if string(command.Operation) != strings.TrimSpace(string(command.Operation)) {
		return errors.New("invalid operation name")
	}
	if len(command.Parameters) > 12 || len(command.Options) > 32 {
		return errors.New("workspace request exceeds bounds")
	}
	if err := validateNamedValues(command.Parameters, command.Options); err != nil {
		return err
	}
	if policy.Mode == ModeRead {
		if !approval.Read || approval.Apply || approval.Confirm != "" || approval.ConfirmDangerous != "" {
			return errors.New("read requires explicit read approval without write confirmation")
		}
		return nil
	}
	if !approval.Apply || approval.Read || approval.Confirm != command.Operation {
		return errors.New("write requires apply approval and matching operation confirmation")
	}
	if policy.Dangerous && approval.ConfirmDangerous != command.Operation {
		return errors.New("dangerous write requires explicit additional confirmation")
	}
	if !policy.Dangerous && approval.ConfirmDangerous != "" {
		return errors.New("unneeded dangerous confirmation")
	}
	return nil
}

func (s *Operations) Execute(ctx context.Context, api API, command OperationCommand, approval OperationApproval) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, ErrContextRequired
	}
	if err := ctx.Err(); err != nil {
		return OperationResult{}, err
	}
	if err := ValidateOperation(api, command, approval); err != nil {
		return OperationResult{}, err
	}
	command = cloneOperationCommand(command)
	var (
		result OperationResult
		err    error
	)
	switch api {
	case APICalendar:
		result, err = s.calendar.ExecuteCalendar(ctx, command)
	case APITasks:
		result, err = s.tasks.ExecuteTasks(ctx, command)
	case APIForms:
		result, err = s.forms.ExecuteForms(ctx, command)
	default:
		return OperationResult{}, errors.New("unsupported workspace provider")
	}
	result.Payload = append([]byte(nil), result.Payload...)
	return result, err
}

func validateNamedValues(parameters []Parameter, options []Option) error {
	seenParameters := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		if !validName(parameter.Name) {
			return errors.New("invalid workspace parameter name")
		}
		if _, exists := seenParameters[parameter.Name]; exists {
			return errors.New("duplicate workspace parameter")
		}
		seenParameters[parameter.Name] = struct{}{}
	}
	seenOptions := make(map[string]struct{}, len(options))
	for _, option := range options {
		if !validName(option.Name) {
			return errors.New("invalid workspace option name")
		}
		if _, exists := seenOptions[option.Name]; exists {
			return errors.New("duplicate workspace option")
		}
		seenOptions[option.Name] = struct{}{}
	}
	return nil
}

func validName(name string) bool {
	return name != "" && name == strings.TrimSpace(name) && !strings.ContainsAny(name, "\r\n\t")
}

func cloneOperationCommand(command OperationCommand) OperationCommand {
	cloned := command
	cloned.Parameters = append([]Parameter(nil), command.Parameters...)
	cloned.Options = make([]Option, len(command.Options))
	for index, option := range command.Options {
		cloned.Options[index] = Option{Name: option.Name, Values: append([]string(nil), option.Values...)}
	}
	cloned.Payload = append([]byte(nil), command.Payload...)
	return cloned
}
