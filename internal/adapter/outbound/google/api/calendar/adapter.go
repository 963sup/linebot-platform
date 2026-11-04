package calendar

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	googlecalendar "google.golang.org/api/calendar/v3"
)

type Adapter struct{ client *googlecalendar.Service }

func New(client *googlecalendar.Service) (*Adapter, error) {
	if client == nil {
		return nil, errors.New("calendar SDK service is required")
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) ExecuteCalendar(ctx context.Context, req workspace.Command) (workspace.Result, error) {
	if ctx == nil {
		return workspace.Result{}, errors.New("workspace context required")
	}
	if err := ctx.Err(); err != nil {
		return workspace.Result{}, err
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
		return workspace.Result{}, errors.New("unsupported calendar SDK operation")
	}
}
