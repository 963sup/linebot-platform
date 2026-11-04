package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	googletasks "google.golang.org/api/tasks/v1"
)

type Adapter struct{ client *googletasks.Service }

func New(client *googletasks.Service) (*Adapter, error) {
	if client == nil {
		return nil, errors.New("tasks SDK service is required")
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) ExecuteTasks(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("workspace context required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch strings.SplitN(req.Operation, ".", 2)[0] {
	case "tasklists":
		return a.executeTasklists(ctx, req)
	case "tasks":
		return a.executeTasks(ctx, req)
	default:
		return nil, errors.New("unsupported tasks SDK operation")
	}
}
