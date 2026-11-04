package outbound

import (
	"context"
	"encoding/json"
)

// WorkspaceRequest carries explicit operation identifiers and JSON-shaped
// arguments. Only the outer SDK adapters may import Google API types.
type WorkspaceRequest struct {
	Operation  string                     `json:"operation"`
	Parameters map[string]json.RawMessage `json:"parameters,omitempty"`
	Options    map[string]json.RawMessage `json:"options,omitempty"`
	Body       json.RawMessage            `json:"body,omitempty"`
}

type GoogleCalendarOperator interface {
	ExecuteCalendar(context.Context, WorkspaceRequest) (json.RawMessage, error)
}

type GoogleTasksOperator interface {
	ExecuteTasks(context.Context, WorkspaceRequest) (json.RawMessage, error)
}
