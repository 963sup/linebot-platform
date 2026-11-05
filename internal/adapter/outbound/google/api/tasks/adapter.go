package tasks

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	googletasks "google.golang.org/api/tasks/v1"
)

type Adapter struct{ client *googletasks.Service }

var _ workspace.TasksGateway = (*Adapter)(nil)

func New(client *googletasks.Service) (*Adapter, error) {
	if client == nil {
		return nil, errors.New("tasks SDK service is required")
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) ExecuteTasks(ctx context.Context, command workspace.OperationCommand) (workspace.OperationResult, error) {
	if ctx == nil {
		return workspace.OperationResult{}, errors.New("workspace context required")
	}
	if err := ctx.Err(); err != nil {
		return workspace.OperationResult{}, err
	}
	req, err := codec.Decode(command)
	if err != nil {
		return workspace.OperationResult{}, err
	}
	switch strings.SplitN(req.Operation, ".", 2)[0] {
	case "tasklists":
		return a.executeTasklists(ctx, req)
	case "tasks":
		return a.executeTasks(ctx, req)
	default:
		return workspace.OperationResult{}, errors.New("unsupported tasks SDK operation")
	}
}
