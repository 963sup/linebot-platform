package calendar

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeFreebusy(ctx context.Context, req workspace.Command) (workspace.Result, error) {
	switch req.Operation {
	case "freebusy.query":
		if err := codec.Validate(req, true, []string{}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		body, err := codec.Body[googlecalendar.FreeBusyRequest](req.Body)
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Freebusy.Query(body)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "freebusy.query", err)
		}
		return codec.Encode(response)

	default:
		return workspace.Result{}, codec.UnsupportedOperation()
	}
}
