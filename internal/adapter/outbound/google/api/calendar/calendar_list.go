package calendar

import (
	"context"
	"encoding/json"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeCalendarList(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage, error) {
	switch req.Operation {
	case "calendarList.delete":
		if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		call := a.client.CalendarList.Delete(param1)
		if err := call.Context(ctx).Do(); err != nil {
			return nil, codec.Wrap("calendar", "calendarList.delete", err)
		}
		return nil, nil

	case "calendarList.get":
		if err := codec.Validate(req, false, []string{"calendarId"}, []string{}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		call := a.client.CalendarList.Get(param1)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "calendarList.get", err)
		}
		return codec.Encode(response)

	case "calendarList.insert":
		if err := codec.Validate(req, true, []string{}, []string{"colorRgbFormat"}); err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.CalendarListEntry](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.CalendarList.Insert(body)
		if _, ok := req.Options["colorRgbFormat"]; ok {
			value0, err := codec.Optional[bool](req.Options, "colorRgbFormat")
			if err != nil {
				return nil, err
			}
			call = call.ColorRgbFormat(value0)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "calendarList.insert", err)
		}
		return codec.Encode(response)

	case "calendarList.list":
		if err := codec.Validate(req, false, []string{}, []string{"maxResults", "minAccessRole", "pageToken", "showDeleted", "showHidden", "showOwnOrganizationOnly", "syncToken"}); err != nil {
			return nil, err
		}
		call := a.client.CalendarList.List()
		if _, ok := req.Options["maxResults"]; ok {
			value0, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return nil, err
			}
			call = call.MaxResults(value0)
		}
		if _, ok := req.Options["minAccessRole"]; ok {
			value1, err := codec.Optional[string](req.Options, "minAccessRole")
			if err != nil {
				return nil, err
			}
			call = call.MinAccessRole(value1)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value2, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return nil, err
			}
			call = call.PageToken(value2)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value3, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return nil, err
			}
			call = call.ShowDeleted(value3)
		}
		if _, ok := req.Options["showHidden"]; ok {
			value4, err := codec.Optional[bool](req.Options, "showHidden")
			if err != nil {
				return nil, err
			}
			call = call.ShowHidden(value4)
		}
		if _, ok := req.Options["showOwnOrganizationOnly"]; ok {
			value5, err := codec.Optional[bool](req.Options, "showOwnOrganizationOnly")
			if err != nil {
				return nil, err
			}
			call = call.ShowOwnOrganizationOnly(value5)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value6, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return nil, err
			}
			call = call.SyncToken(value6)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "calendarList.list", err)
		}
		return codec.Encode(response)

	case "calendarList.patch":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{"colorRgbFormat"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.CalendarListEntry](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.CalendarList.Patch(param1, body)
		if _, ok := req.Options["colorRgbFormat"]; ok {
			value0, err := codec.Optional[bool](req.Options, "colorRgbFormat")
			if err != nil {
				return nil, err
			}
			call = call.ColorRgbFormat(value0)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "calendarList.patch", err)
		}
		return codec.Encode(response)

	case "calendarList.update":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{"colorRgbFormat"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.CalendarListEntry](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.CalendarList.Update(param1, body)
		if _, ok := req.Options["colorRgbFormat"]; ok {
			value0, err := codec.Optional[bool](req.Options, "colorRgbFormat")
			if err != nil {
				return nil, err
			}
			call = call.ColorRgbFormat(value0)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "calendarList.update", err)
		}
		return codec.Encode(response)

	case "calendarList.watch":
		if err := codec.Validate(req, true, []string{}, []string{"maxResults", "minAccessRole", "pageToken", "showDeleted", "showHidden", "showOwnOrganizationOnly", "syncToken"}); err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.Channel](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.CalendarList.Watch(body)
		if _, ok := req.Options["maxResults"]; ok {
			value0, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return nil, err
			}
			call = call.MaxResults(value0)
		}
		if _, ok := req.Options["minAccessRole"]; ok {
			value1, err := codec.Optional[string](req.Options, "minAccessRole")
			if err != nil {
				return nil, err
			}
			call = call.MinAccessRole(value1)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value2, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return nil, err
			}
			call = call.PageToken(value2)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value3, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return nil, err
			}
			call = call.ShowDeleted(value3)
		}
		if _, ok := req.Options["showHidden"]; ok {
			value4, err := codec.Optional[bool](req.Options, "showHidden")
			if err != nil {
				return nil, err
			}
			call = call.ShowHidden(value4)
		}
		if _, ok := req.Options["showOwnOrganizationOnly"]; ok {
			value5, err := codec.Optional[bool](req.Options, "showOwnOrganizationOnly")
			if err != nil {
				return nil, err
			}
			call = call.ShowOwnOrganizationOnly(value5)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value6, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return nil, err
			}
			call = call.SyncToken(value6)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "calendarList.watch", err)
		}
		return codec.Encode(response)

	default:
		return nil, codec.UnsupportedOperation()
	}
}
