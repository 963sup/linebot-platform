package calendar

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeSettings(ctx context.Context, req workspace.Command) (workspace.Result, error) {
	switch req.Operation {
	case "settings.get":
		if err := codec.Validate(req, false, []string{"setting"}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "setting")
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Settings.Get(param1)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "settings.get", err)
		}
		return codec.Encode(response)

	case "settings.list":
		if err := codec.Validate(req, false, []string{}, []string{"maxResults", "pageToken", "syncToken"}); err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Settings.List()
		if _, ok := req.Options["maxResults"]; ok {
			value0, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return workspace.Result{}, err
			}
			call = call.MaxResults(value0)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value1, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return workspace.Result{}, err
			}
			call = call.PageToken(value1)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value2, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return workspace.Result{}, err
			}
			call = call.SyncToken(value2)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "settings.list", err)
		}
		return codec.Encode(response)

	case "settings.watch":
		if err := codec.Validate(req, true, []string{}, []string{"maxResults", "pageToken", "syncToken"}); err != nil {
			return workspace.Result{}, err
		}
		body, err := codec.Body[googlecalendar.Channel](req.Body)
		if err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Settings.Watch(body)
		if _, ok := req.Options["maxResults"]; ok {
			value0, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return workspace.Result{}, err
			}
			call = call.MaxResults(value0)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value1, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return workspace.Result{}, err
			}
			call = call.PageToken(value1)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value2, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return workspace.Result{}, err
			}
			call = call.SyncToken(value2)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "settings.watch", err)
		}
		return codec.Encode(response)

	default:
		return workspace.Result{}, codec.UnsupportedOperation()
	}
}
