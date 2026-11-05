package main

import (
	"context"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	"github.com/963sup/linebot-platform/internal/bootstrap"
	"golang.org/x/oauth2"
)

// workspaceExecutor keeps the consumer-owned Workspace contract visible at
// the composition boundary while allowing focused tests without Google SDKs.
type workspaceExecutor func(context.Context, workspaceautomation.API, workspaceautomation.OperationCommand, workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error)

var _ cli.WorkspaceRuntime = workspaceExecutor(nil)

func (execute workspaceExecutor) Execute(ctx context.Context, api workspaceautomation.API, request workspaceautomation.OperationCommand, approval workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error) {
	result, err := execute(ctx, api, request, approval)
	if err != nil {
		return workspaceautomation.OperationResult{}, err
	}
	return result, nil
}

func newWorkspaceRuntime(ctx context.Context, source oauth2.TokenSource) (cli.WorkspaceRuntime, error) {
	operations, err := bootstrap.NewWorkspaceOperations(ctx, source)
	if err != nil {
		return nil, err
	}
	return workspaceExecutor(operations.Execute), nil
}
