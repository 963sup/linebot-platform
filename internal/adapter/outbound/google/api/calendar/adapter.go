package calendar

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	googlecalendar "google.golang.org/api/calendar/v3"
)

type Adapter struct{ client *googlecalendar.Service }

var _ workspace.CalendarGateway = (*Adapter)(nil)

func New(client *googlecalendar.Service) (*Adapter, error) {
	if client == nil {
		return nil, errors.New("calendar SDK service is required")
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) ExecuteCalendar(ctx context.Context, command workspace.OperationCommand) (workspace.OperationResult, error) {
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
	case "acl":
		return a.executeAcl(ctx, req)
	case "calendarList":
		return a.executeCalendarList(ctx, req)
	case "calendars":
		return a.executeCalendars(ctx, req)
	case "channels":
		return a.executeChannels(ctx, req)
	case "colors":
		return a.executeColors(ctx, req)
	case "events":
		return a.executeEvents(ctx, req)
	case "freebusy":
		return a.executeFreebusy(ctx, req)
	case "settings":
		return a.executeSettings(ctx, req)
	default:
		return workspace.OperationResult{}, errors.New("unsupported calendar SDK operation")
	}
}
