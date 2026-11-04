package main

import (
	"context"
	"encoding/json"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	"github.com/963sup/linebot-platform/internal/bootstrap"
	"golang.org/x/oauth2"
)

// workspaceExecutor adapts the use case's named JSON result to the CLI's byte
// contract without re-encoding the payload or changing authorization behavior.
type workspaceExecutor func(context.Context, string, outbound.WorkspaceRequest, workspace.Approval) (json.RawMessage, error)

var _ cli.WorkspaceRuntime = workspaceExecutor(nil)

func (execute workspaceExecutor) Execute(ctx context.Context, action string, request outbound.WorkspaceRequest, approval workspace.Approval) ([]byte, error) {
	return execute(ctx, action, request, approval)
}

func newWorkspaceRuntime(ctx context.Context, source oauth2.TokenSource) (cli.WorkspaceRuntime, error) {
	operations, err := bootstrap.NewWorkspaceOperations(ctx, source)
	if err != nil {
		return nil, err
	}
	return workspaceExecutor(operations.Execute), nil
}
