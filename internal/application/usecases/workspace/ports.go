package workspace

import "context"

// Command is the application-owned structured representation of one statically
// allowlisted Workspace operation. JSON decoding and Google SDK mapping belong
// to the inbound and outbound adapters respectively.
type Command struct {
	Operation  string
	Parameters map[string]any
	Options    map[string]any
	Body       any
}

type Result struct {
	Value any
}

type CalendarOperator interface {
	ExecuteCalendar(context.Context, Command) (Result, error)
}

type TasksOperator interface {
	ExecuteTasks(context.Context, Command) (Result, error)
}

type FormsOperator interface {
	ExecuteForms(context.Context, Command) (Result, error)
}
