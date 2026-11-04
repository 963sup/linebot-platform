package main

import (
	"context"
	"encoding/json"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	"github.com/963sup/linebot-platform/internal/bootstrap"
	"golang.org/x/oauth2"
)

// workspaceExecutor adapts the use case's named JSON result to the CLI's byte
// contract without re-encoding the payload or changing authorization behavior.
type workspaceExecutor func(context.Context, string, workspace.Command, workspace.Approval) (workspace.Result, error)

var _ cli.WorkspaceRuntime = workspaceExecutor(nil)

func (execute workspaceExecutor) Execute(ctx context.Context, action string, request workspace.Command, approval workspace.Approval) ([]byte, error) {
	result, err := execute(ctx, action, request, approval)
	if err != nil {
		return nil, err
	}
	if result.Value == nil {
		return nil, nil
	}
	return json.Marshal(result.Value)
}

func newWorkspaceRuntime(ctx context.Context, source oauth2.TokenSource) (cli.WorkspaceRuntime, error) {
	operations, err := bootstrap.NewWorkspaceOperations(ctx, source)
	if err != nil {
		return nil, err
	}
	return workspaceExecutor(operations.Execute), nil
}
