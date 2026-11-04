package calendar

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeCalendars(ctx context.Context, req workspace.Command) (workspace.Result, error) {
	switch req.Operation {
	case "calendars.clear":
		if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Calendars.Clear(param1)
		if err := call.Context(ctx).Do(); err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "calendars.clear", err)
		}
		return workspace.Result{}, nil

	case "calendars.delete":
		if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Calendars.Delete(param1)
		if err := call.Context(ctx).Do(); err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "calendars.delete", err)
		}
		return workspace.Result{}, nil

	case "calendars.get":
		if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Calendars.Get(param1)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "calendars.get", err)
		}
		return codec.Encode(response)

	case "calendars.insert":
		if err := codec.Validate(req, true, []string{}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		body, err := codec.Body[googlecalendar.Calendar](req.Body)
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Calendars.Insert(body)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "calendars.insert", err)
		}
		return codec.Encode(response)

	case "calendars.patch":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return workspace.Result{}, err
		}
		body, err := codec.Body[googlecalendar.Calendar](req.Body)
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Calendars.Patch(param1, body)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "calendars.patch", err)
		}
		return codec.Encode(response)

	case "calendars.transferOwnership":
		if err := codec.Validate(req, false, []string{"calendarId", "newDataOwner", "useAdminAccess"}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return workspace.Result{}, err
		}
		param2, err := codec.Required[string](req.Parameters, "newDataOwner")
		if err != nil {
			return workspace.Result{}, err
		}
		param3, err := codec.Required[bool](req.Parameters, "useAdminAccess")
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Calendars.TransferOwnership(param1, param2, param3)
		if err := call.Context(ctx).Do(); err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "calendars.transferOwnership", err)
		}
		return workspace.Result{}, nil

	case "calendars.update":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return workspace.Result{}, err
		}
		body, err := codec.Body[googlecalendar.Calendar](req.Body)
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Calendars.Update(param1, body)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "calendars.update", err)
		}
		return codec.Encode(response)

	default:
		return workspace.Result{}, codec.UnsupportedOperation()
	}
}
