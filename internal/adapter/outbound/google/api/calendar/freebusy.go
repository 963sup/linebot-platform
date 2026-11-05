package calendar

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeFreebusy(ctx context.Context, req codec.Request) (workspace.OperationResult, error) {
	switch req.Operation {
	case "freebusy.query":
		if err := codec.Validate(req, true, []string{}, []string{}); err != nil {
			return workspace.OperationResult{}, err
		}
		body, err := codec.Body[googlecalendar.FreeBusyRequest](req.Body)
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Freebusy.Query(body)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("calendar", "freebusy.query", err)
		}
		return codec.Encode(response)

	default:
		return workspace.OperationResult{}, codec.UnsupportedOperation()
	}
}
