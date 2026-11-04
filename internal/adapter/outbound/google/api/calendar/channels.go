package calendar

import (
	"context"
	"encoding/json"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeChannels(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage, error) {
	switch req.Operation {
	case "channels.stop":
		if err := codec.Validate(req, true, []string{}, []string{}); err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.Channel](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Channels.Stop(body)
		if err := call.Context(ctx).Do(); err != nil {
			return nil, codec.Wrap("calendar", "channels.stop", err)
		}
		return nil, nil

	default:
		return nil, codec.UnsupportedOperation()
	}
}
