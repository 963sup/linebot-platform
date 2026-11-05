// Package events defines provider-neutral workspace automation facts.
package events

import valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"

// AutomationEvent is a marker for workspace automation domain events.
type AutomationEvent interface{ AutomationEvent() }

type AutomationRequested struct {
	ID     valueobjects.AutomationRequestID
	Action valueobjects.Action
}

func (AutomationRequested) AutomationEvent() {}

type AutomationApproved struct {
	ID       valueobjects.AutomationRequestID
	Approval valueobjects.Approval
}

func (AutomationApproved) AutomationEvent() {}

type AutomationExecutionStarted struct {
	ID valueobjects.AutomationRequestID
}

func (AutomationExecutionStarted) AutomationEvent() {}

type AutomationSucceeded struct {
	ID     valueobjects.AutomationRequestID
	Output valueobjects.Output
}

func (AutomationSucceeded) AutomationEvent() {}

type AutomationFailed struct {
	ID     valueobjects.AutomationRequestID
	Output valueobjects.Output
}

func (AutomationFailed) AutomationEvent() {}

type AutomationOutcomeUnknown struct {
	ID     valueobjects.AutomationRequestID
	Output valueobjects.Output
}

func (AutomationOutcomeUnknown) AutomationEvent() {}
