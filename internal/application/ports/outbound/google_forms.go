package outbound

import (
	"context"
	"encoding/json"
)

type GoogleFormsOperator interface {
	ExecuteForms(context.Context, WorkspaceRequest) (json.RawMessage, error)
}
