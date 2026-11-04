package calendar

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeChannels(ctx context.Context, req workspace.Command) (workspace.Result, error) {
	switch req.Operation {
	case "channels.stop":
		if err := codec.Validate(req, true, []string{}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		body, err := codec.Body[googlecalendar.Channel](req.Body)
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Channels.Stop(body)
		if err := call.Context(ctx).Do(); err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "channels.stop", err)
		}
		return workspace.Result{}, nil

	default:
		return workspace.Result{}, codec.UnsupportedOperation()
	}
}
