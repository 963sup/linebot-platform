// Package aggregates owns workspace automation consistency boundaries.
package aggregates

import (
	"errors"

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
	}, nil
}

// RestoreAutomationRequest rehydrates a valid persisted state.
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
	return nil
}

func (a *AutomationRequest) valid() bool {
	return a != nil && a.id.Valid() && a.action.Valid() && a.requiredScopes.Valid() && a.status.Valid()
}
