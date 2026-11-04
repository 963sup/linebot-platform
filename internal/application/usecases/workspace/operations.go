package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// Policy is intentionally static and pinned to the audited Google Discovery
// snapshot. No reflection or dynamic Google API invoker is involved.
type Policy struct {
	API       string
	Operation string
	Mode      string
	Dangerous bool
}

var calendarPolicies = []Policy{
	{API: "calendar", Operation: "acl.delete", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "acl.get", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "acl.insert", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "acl.list", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "acl.patch", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "acl.update", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "acl.watch", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "calendarList.delete", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "calendarList.get", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "calendarList.insert", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "calendarList.list", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "calendarList.patch", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "calendarList.update", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "calendarList.watch", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "calendars.clear", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "calendars.delete", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "calendars.get", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "calendars.insert", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "calendars.patch", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "calendars.transferOwnership", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "calendars.update", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "channels.stop", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "colors.get", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "events.delete", Mode: "write", Dangerous: true},
	{API: "calendar", Operation: "events.get", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "events.import", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "events.insert", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "events.instances", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "events.list", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "events.move", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "events.patch", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "events.quickAdd", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "events.update", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "events.watch", Mode: "write", Dangerous: false},
	{API: "calendar", Operation: "freebusy.query", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "settings.get", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "settings.list", Mode: "read", Dangerous: false},
	{API: "calendar", Operation: "settings.watch", Mode: "write", Dangerous: false},
}

var tasksPolicies = []Policy{
	{API: "tasks", Operation: "tasklists.delete", Mode: "write", Dangerous: true},
	{API: "tasks", Operation: "tasklists.get", Mode: "read", Dangerous: false},
	{API: "tasks", Operation: "tasklists.insert", Mode: "write", Dangerous: false},
	{API: "tasks", Operation: "tasklists.list", Mode: "read", Dangerous: false},
	{API: "tasks", Operation: "tasklists.patch", Mode: "write", Dangerous: false},
	{API: "tasks", Operation: "tasklists.update", Mode: "write", Dangerous: false},
	{API: "tasks", Operation: "tasks.clear", Mode: "write", Dangerous: true},
	{API: "tasks", Operation: "tasks.delete", Mode: "write", Dangerous: true},
	{API: "tasks", Operation: "tasks.get", Mode: "read", Dangerous: false},
	{API: "tasks", Operation: "tasks.insert", Mode: "write", Dangerous: false},
	{API: "tasks", Operation: "tasks.list", Mode: "read", Dangerous: false},
	{API: "tasks", Operation: "tasks.move", Mode: "write", Dangerous: false},
	{API: "tasks", Operation: "tasks.patch", Mode: "write", Dangerous: false},
	{API: "tasks", Operation: "tasks.update", Mode: "write", Dangerous: false},
}

var formsPolicies = []Policy{
	{API: "forms", Operation: "forms.batchUpdate", Mode: "write", Dangerous: true},
	{API: "forms", Operation: "forms.create", Mode: "write", Dangerous: false},
	{API: "forms", Operation: "forms.get", Mode: "read", Dangerous: false},
	{API: "forms", Operation: "forms.setPublishSettings", Mode: "write", Dangerous: true},
	{API: "forms", Operation: "forms.responses.get", Mode: "read", Dangerous: false},
	{API: "forms", Operation: "forms.responses.list", Mode: "read", Dangerous: false},
	{API: "forms", Operation: "forms.watches.create", Mode: "write", Dangerous: false},
	{API: "forms", Operation: "forms.watches.delete", Mode: "write", Dangerous: true},
	{API: "forms", Operation: "forms.watches.list", Mode: "read", Dangerous: false},
	{API: "forms", Operation: "forms.watches.renew", Mode: "write", Dangerous: false},
}

func Supported() []Policy {
	all := make([]Policy, 0, len(calendarPolicies)+len(tasksPolicies)+len(formsPolicies))
	all = append(all, calendarPolicies...)
	all = append(all, tasksPolicies...)
	return append(all, formsPolicies...)
}

func Lookup(api, operation string) (Policy, bool) {
	for _, p := range Supported() {
		if p.API == api && p.Operation == operation {
			return p, true
		}
	}
	return Policy{}, false
}

type Approval struct {
	Read             bool
	Apply            bool
	Confirm          string
	ConfirmDangerous string
}

// Operations orchestrates provider-independent authorization; the adapters
// own SDK request and response mapping. This is never routed from the
// publicly accessible LINE webhook.
type Operations struct {
	calendar outbound.GoogleCalendarOperator
	tasks    outbound.GoogleTasksOperator
	forms    outbound.GoogleFormsOperator
}

func NewOperations(cal outbound.GoogleCalendarOperator, task outbound.GoogleTasksOperator) (*Operations, error) {
	if cal == nil || task == nil {
		return nil, errors.New("Calendar and Tasks operators are required")
	}
	return &Operations{calendar: cal, tasks: task}, nil
}

// NewOperationsWithForms leaves existing Calendar/Tasks constructors intact.
func NewOperationsWithForms(cal outbound.GoogleCalendarOperator, tasks outbound.GoogleTasksOperator, forms outbound.GoogleFormsOperator) (*Operations, error) {
	if forms == nil {
		return nil, errors.New("Forms operator is required")
	}
	ops, err := NewOperations(cal, tasks)
	if err != nil {
		return nil, err
	}
	ops.forms = forms
	return ops, nil
}

func Validate(api string, req outbound.WorkspaceRequest, approval Approval) error {
	p, ok := Lookup(api, req.Operation)
	if !ok {
		return errors.New("unsupported Google Workspace operation")
	}
	if strings.TrimSpace(req.Operation) != req.Operation {
		return errors.New("invalid operation name")
	}
	if len(req.Parameters) > 12 || len(req.Options) > 32 || len(req.Body) > 1<<20 {
		return errors.New("workspace request exceeds bounds")
	}
	if p.Mode == "read" {
		if !approval.Read || approval.Apply || approval.Confirm != "" || approval.ConfirmDangerous != "" {
			return errors.New("read requires explicit --read without write confirmation")
		}
	} else {
		if !approval.Apply || approval.Read || approval.Confirm != req.Operation {
			return errors.New("write requires --apply --confirm OPERATION")
		}
		if p.Dangerous && approval.ConfirmDangerous != req.Operation {
			return errors.New("dangerous write requires explicit additional confirmation")
		}
		if !p.Dangerous && approval.ConfirmDangerous != "" {
			return errors.New("unneeded dangerous confirmation")
		}
	}
	for _, v := range req.Parameters {
		if !json.Valid(v) {
			return errors.New("invalid operation parameter")
		}
	}
	for _, v := range req.Options {
		if !json.Valid(v) {
			return errors.New("invalid operation option")
		}
	}
	if len(req.Body) > 0 && !json.Valid(req.Body) {
		return errors.New("invalid operation body")
	}
	return nil
}

func (s *Operations) Execute(ctx context.Context, api string, req outbound.WorkspaceRequest, approval Approval) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := Validate(api, req, approval); err != nil {
		return nil, err
	}
	switch api {
	case "calendar":
		return s.calendar.ExecuteCalendar(ctx, req)
	case "tasks":
		return s.tasks.ExecuteTasks(ctx, req)
	case "forms":
		if s.forms == nil {
			return nil, errors.New("Forms adapter not configured")
		}
		return s.forms.ExecuteForms(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported workspace provider")
	}
}
