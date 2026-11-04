package tasks

import (
    "context"
    "encoding/json"
    "github.com/963sup/linebot-platform/internal/application/ports/outbound"
    "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
    googletasks "google.golang.org/api/tasks/v1"
)

func (a *Adapter) executeTasklists(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage,error) {
    switch req.Operation {
case "tasklists.delete":
        if err := codec.Validate(req, false, []string{"tasklist"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "tasklist")
        if err != nil {return nil, err}
        call := a.client.Tasklists.Delete(param1)
        if err := call.Context(ctx).Do(); err != nil {return nil, codec.Wrap("tasks", "tasklists.delete",err)}
        return nil, nil

case "tasklists.get":
        if err := codec.Validate(req, false, []string{"tasklist"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "tasklist")
        if err != nil {return nil, err}
        call := a.client.Tasklists.Get(param1)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("tasks", "tasklists.get",err)}
        return codec.Encode(response)

case "tasklists.insert":
        if err := codec.Validate(req, true, []string{}, []string{}); err != nil {return nil, err}
        body, err := codec.Body[googletasks.TaskList](req.Body)
        if err != nil {return nil, err}
        call := a.client.Tasklists.Insert(body)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("tasks", "tasklists.insert",err)}
        return codec.Encode(response)

case "tasklists.list":
        if err := codec.Validate(req, false, []string{}, []string{"maxResults", "pageToken"}); err != nil {return nil, err}
        call := a.client.Tasklists.List()
        if _, ok := req.Options["maxResults"]; ok {
          value0, err := codec.Optional[int64](req.Options, "maxResults")
          if err != nil {return nil, err}
          call = call.MaxResults(value0)
        }
        if _, ok := req.Options["pageToken"]; ok {
          value1, err := codec.Optional[string](req.Options, "pageToken")
          if err != nil {return nil, err}
          call = call.PageToken(value1)
        }
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("tasks", "tasklists.list",err)}
        return codec.Encode(response)

case "tasklists.patch":
        if err := codec.Validate(req, true, []string{"tasklist"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "tasklist")
        if err != nil {return nil, err}
        body, err := codec.Body[googletasks.TaskList](req.Body)
        if err != nil {return nil, err}
        call := a.client.Tasklists.Patch(param1, body)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("tasks", "tasklists.patch",err)}
        return codec.Encode(response)

case "tasklists.update":
        if err := codec.Validate(req, true, []string{"tasklist"}, []string{}); err != nil {return nil, err}
        param1, err := codec.Required[string](req.Parameters, "tasklist")
        if err != nil {return nil, err}
        body, err := codec.Body[googletasks.TaskList](req.Body)
        if err != nil {return nil, err}
        call := a.client.Tasklists.Update(param1, body)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("tasks", "tasklists.update",err)}
        return codec.Encode(response)

    default:
        return nil, codec.UnsupportedOperation()
    }
}
