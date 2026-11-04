package calendar

import (
    "context"
    "encoding/json"
    "github.com/963sup/linebot-platform/internal/application/ports/outbound"
    "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
    googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeCalendars(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage,error) {
    switch req.Operation {
case "calendars.clear":
        if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "calendarId")
        if err != nil {return nil, err}
        call := a.client.Calendars.Clear(param1)
        if err := call.Context(ctx).Do(); err != nil {return nil, codec.Wrap("calendar", "calendars.clear",err)}
        return nil, nil

case "calendars.delete":
        if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "calendarId")
        if err != nil {return nil, err}
        call := a.client.Calendars.Delete(param1)
        if err := call.Context(ctx).Do(); err != nil {return nil, codec.Wrap("calendar", "calendars.delete",err)}
        return nil, nil

case "calendars.get":
        if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "calendarId")
        if err != nil {return nil, err}
        call := a.client.Calendars.Get(param1)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("calendar", "calendars.get",err)}
        return codec.Encode(response)

case "calendars.insert":
        if err := codec.Validate(req, true, []string{}, []string{}); err != nil {return nil, err}
        body, err := codec.Body[googlecalendar.Calendar](req.Body)
        if err != nil {return nil, err}
        call := a.client.Calendars.Insert(body)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("calendar", "calendars.insert",err)}
        return codec.Encode(response)

case "calendars.patch":
        if err := codec.Validate(req, true, []string{"calendarId"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "calendarId")
        if err != nil {return nil, err}
        body, err := codec.Body[googlecalendar.Calendar](req.Body)
        if err != nil {return nil, err}
        call := a.client.Calendars.Patch(param1, body)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("calendar", "calendars.patch",err)}
        return codec.Encode(response)

case "calendars.transferOwnership":
        if err := codec.Validate(req, false, []string{"calendarId", "newDataOwner", "useAdminAccess"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "calendarId")
        if err != nil {return nil, err}
        param2, err := codec.Required[string](req.Parameters, "newDataOwner")
        if err != nil {return nil, err}
        param3, err := codec.Required[bool](req.Parameters, "useAdminAccess")
        if err != nil {return nil, err}
        call := a.client.Calendars.TransferOwnership(param1, param2, param3)
        if err := call.Context(ctx).Do(); err != nil {return nil, codec.Wrap("calendar", "calendars.transferOwnership",err)}
        return nil, nil

case "calendars.update":
        if err := codec.Validate(req, true, []string{"calendarId"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "calendarId")
        if err != nil {return nil, err}
        body, err := codec.Body[googlecalendar.Calendar](req.Body)
        if err != nil {return nil, err}
        call := a.client.Calendars.Update(param1, body)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("calendar", "calendars.update",err)}
        return codec.Encode(response)

    default:
        return nil, codec.UnsupportedOperation()
    }
}
