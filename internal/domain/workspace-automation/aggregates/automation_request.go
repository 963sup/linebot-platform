// Package aggregates owns workspace automation consistency boundaries.
package aggregates

import (
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

var (
	ErrInvalidAutomationRequest = errors.New("invalid automation request")
	ErrInvalidTransition        = errors.New("invalid automation request transition")
	ErrApprovalRequired         = errors.New("automation request approval required")
)

// AutomationRequest is the aggregate root for one approved workspace action.
// Provider credentials, SDK requests, and transport metadata never belong here.
type AutomationRequest struct {
	id             valueobjects.AutomationRequestID
	action         valueobjects.Action
	requiredScopes valueobjects.RequiredScopes
	approval       valueobjects.Approval
	output         valueobjects.Output
	status         valueobjects.Status
	revision       uint64
	pending        []events.AutomationEvent
}

func NewAutomationRequest(id valueobjects.AutomationRequestID, action valueobjects.Action, scopes valueobjects.RequiredScopes) (AutomationRequest, error) {
	if !id.Valid() || !action.Valid() || !scopes.Valid() {
		return AutomationRequest{}, ErrInvalidAutomationRequest
	}
	return AutomationRequest{
		id:             id,
		action:         action,
		requiredScopes: scopes,
		status:         valueobjects.StatusRequested,
		revision:       1,
		pending:        []events.AutomationEvent{events.AutomationRequested{ID: id, Action: action}},
	}, nil
}

// RestoreAutomationRequest rehydrates a valid persisted state without raising events.
func RestoreAutomationRequest(id valueobjects.AutomationRequestID, action valueobjects.Action, scopes valueobjects.RequiredScopes, approval valueobjects.Approval, output valueobjects.Output, status valueobjects.Status, revision uint64) (AutomationRequest, error) {
	if !id.Valid() || !action.Valid() || !scopes.Valid() || !status.Valid() || revision == 0 {
		return AutomationRequest{}, ErrInvalidAutomationRequest
	}
	switch status {
	case valueobjects.StatusRequested:
		if approval.Valid() || output.Valid() {
			return AutomationRequest{}, ErrInvalidAutomationRequest
		}
	case valueobjects.StatusApproved, valueobjects.StatusExecuting:
		if !approval.Valid() || output.Valid() {
			return AutomationRequest{}, ErrInvalidAutomationRequest
		}
	default:
		if !approval.Valid() || !output.Valid() {
			return AutomationRequest{}, ErrInvalidAutomationRequest
		}
	}
	return AutomationRequest{id: id, action: action, requiredScopes: scopes, approval: approval, output: output, status: status, revision: revision}, nil
}

func (a AutomationRequest) ID() valueobjects.AutomationRequestID { return a.id }
func (a AutomationRequest) Action() valueobjects.Action          { return a.action }
func (a AutomationRequest) RequiredScopes() []string             { return a.requiredScopes.Values() }
func (a AutomationRequest) Approval() valueobjects.Approval      { return a.approval }
func (a AutomationRequest) Output() valueobjects.Output          { return a.output }
func (a AutomationRequest) Status() valueobjects.Status          { return a.status }
func (a AutomationRequest) Revision() uint64                     { return a.revision }

// PendingEvents returns a detached event snapshot for publication after persistence.
func (a AutomationRequest) PendingEvents() []events.AutomationEvent {
	return append([]events.AutomationEvent(nil), a.pending...)
}

// AcknowledgeEvents clears modeled-only events after aggregate state has been
// saved successfully. This context does not publish integration events yet.
func (a *AutomationRequest) AcknowledgeEvents() {
	if a != nil {
		a.pending = nil
	}
}

func (a *AutomationRequest) Approve(approval valueobjects.Approval) error {
	if !a.valid() || !approval.Valid() {
		return ErrInvalidAutomationRequest
	}
	if a.status != valueobjects.StatusRequested {
		return ErrInvalidTransition
	}
	a.approval = approval
	a.status = valueobjects.StatusApproved
	a.revision++
	a.pending = append(a.pending, events.AutomationApproved{ID: a.id, Approval: approval})
	return nil
}

func (a *AutomationRequest) StartExecution() error {
	if !a.valid() {
		return ErrInvalidAutomationRequest
	}
	if a.status == valueobjects.StatusRequested {
		return ErrApprovalRequired
	}
	if a.status != valueobjects.StatusApproved {
		return ErrInvalidTransition
	}
	a.status = valueobjects.StatusExecuting
	a.revision++
	a.pending = append(a.pending, events.AutomationExecutionStarted{ID: a.id})
	return nil
}

func (a *AutomationRequest) Succeed(output valueobjects.Output) error {
	return a.finish(valueobjects.StatusSucceeded, output)
}

func (a *AutomationRequest) Fail(output valueobjects.Output) error {
	return a.finish(valueobjects.StatusFailed, output)
}

func (a *AutomationRequest) MarkOutcomeUnknown(output valueobjects.Output) error {
	return a.finish(valueobjects.StatusUnknown, output)
}

func (a *AutomationRequest) finish(status valueobjects.Status, output valueobjects.Output) error {
	if !a.valid() || !output.Valid() {
		return ErrInvalidAutomationRequest
	}
	if a.status != valueobjects.StatusExecuting || !status.Terminal() {
		return ErrInvalidTransition
	}
	a.status = status
	a.output = output
	a.revision++
	switch status {
	case valueobjects.StatusSucceeded:
		a.pending = append(a.pending, events.AutomationSucceeded{ID: a.id, Output: output})
	case valueobjects.StatusFailed:
		a.pending = append(a.pending, events.AutomationFailed{ID: a.id, Output: output})
	case valueobjects.StatusUnknown:
		a.pending = append(a.pending, events.AutomationOutcomeUnknown{ID: a.id, Output: output})
	}
	return nil
}

func (a *AutomationRequest) valid() bool {
	return a != nil && a.id.Valid() && a.action.Valid() && a.requiredScopes.Valid() && a.status.Valid()
}
