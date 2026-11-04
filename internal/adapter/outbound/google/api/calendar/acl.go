package calendar

import (
	"context"
	"encoding/json"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeAcl(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage, error) {
	switch req.Operation {
	case "acl.delete":
		if err := codec.Validate(req, false, []string{"calendarId", "ruleId"}, []string{}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "ruleId")
		if err != nil {
			return nil, err
		}
		call := a.client.Acl.Delete(param1, param2)
		if err := call.Context(ctx).Do(); err != nil {
			return nil, codec.Wrap("calendar", "acl.delete", err)
		}
		return nil, nil

	case "acl.get":
		if err := codec.Validate(req, false, []string{"calendarId", "ruleId"}, []string{}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "ruleId")
		if err != nil {
			return nil, err
		}
		call := a.client.Acl.Get(param1, param2)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "acl.get", err)
		}
		return codec.Encode(response)

	case "acl.insert":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{"sendNotifications"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.AclRule](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Acl.Insert(param1, body)
		if _, ok := req.Options["sendNotifications"]; ok {
			value0, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value0)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "acl.insert", err)
		}
		return codec.Encode(response)

	case "acl.list":
		if err := codec.Validate(req, false, []string{"calendarId"}, []string{"maxResults", "pageToken", "showDeleted", "syncToken"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		call := a.client.Acl.List(param1)
		if _, ok := req.Options["maxResults"]; ok {
			value0, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return nil, err
			}
			call = call.MaxResults(value0)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value1, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return nil, err
			}
			call = call.PageToken(value1)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value2, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return nil, err
			}
			call = call.ShowDeleted(value2)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value3, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return nil, err
			}
			call = call.SyncToken(value3)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "acl.list", err)
		}
		return codec.Encode(response)

	case "acl.patch":
		if err := codec.Validate(req, true, []string{"calendarId", "ruleId"}, []string{"sendNotifications"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "ruleId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.AclRule](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Acl.Patch(param1, param2, body)
		if _, ok := req.Options["sendNotifications"]; ok {
			value0, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value0)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "acl.patch", err)
		}
		return codec.Encode(response)

	case "acl.update":
		if err := codec.Validate(req, true, []string{"calendarId", "ruleId"}, []string{"sendNotifications"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "ruleId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.AclRule](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Acl.Update(param1, param2, body)
		if _, ok := req.Options["sendNotifications"]; ok {
			value0, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value0)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "acl.update", err)
		}
		return codec.Encode(response)

	case "acl.watch":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{"maxResults", "pageToken", "showDeleted", "syncToken"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.Channel](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Acl.Watch(param1, body)
		if _, ok := req.Options["maxResults"]; ok {
			value0, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return nil, err
			}
			call = call.MaxResults(value0)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value1, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return nil, err
			}
			call = call.PageToken(value1)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value2, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return nil, err
			}
			call = call.ShowDeleted(value2)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value3, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return nil, err
			}
			call = call.SyncToken(value3)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "acl.watch", err)
		}
		return codec.Encode(response)

	default:
		return nil, codec.UnsupportedOperation()
	}
}
