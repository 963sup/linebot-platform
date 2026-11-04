package bootstrap

import (
	"context"
	"errors"

	caladapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/calendar"
	formadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/forms"
	taskadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/tasks"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	api "github.com/963sup/linebot-platform/internal/infrastructure/google/api"
	"golang.org/x/oauth2"
)

// NewWorkspaceOperations composes Google Workspace only with an explicitly
// provided user-scoped OAuth TokenSource. It is never part of the public LINE
// webhook's dependency graph.
func NewWorkspaceOperations(ctx context.Context, source oauth2.TokenSource) (*workspace.Operations, error) {
	if ctx == nil || source == nil {
		return nil, errors.New("authorized Workspace context and TokenSource required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client := oauth2.NewClient(ctx, source)
	calendarClient, err := api.NewCalendar(ctx, client)
	if err != nil {
		return nil, err
	}
	tasksClient, err := api.NewTasks(ctx, client)
	if err != nil {
		return nil, err
	}
	calendarAdapter, err := caladapter.New(calendarClient)
	if err != nil {
		return nil, err
	}
	tasksAdapter, err := taskadapter.New(tasksClient)
	if err != nil {
		return nil, err
	}
	formsClient, err := api.NewForms(ctx, client)
	if err != nil {
		return nil, err
	}
	formsAdapter, err := formadapter.New(formsClient)
	if err != nil {
		return nil, err
	}
	return workspace.NewOperationsWithForms(calendarAdapter, tasksAdapter, formsAdapter)
}
