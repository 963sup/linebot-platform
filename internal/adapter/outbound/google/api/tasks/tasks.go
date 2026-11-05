package tasks

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	googletasks "google.golang.org/api/tasks/v1"
)

func (a *Adapter) executeTasks(ctx context.Context, req codec.Request) (workspace.OperationResult, error) {
	switch req.Operation {
	case "tasks.clear":
		if err := codec.Validate(req, false, []string{"tasklist"}, []string{}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.Clear(param1)
		if err := call.Context(ctx).Do(); err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.clear", err)
		}
		return workspace.OperationResult{}, nil

	case "tasks.delete":
		if err := codec.Validate(req, false, []string{"tasklist", "task"}, []string{}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		param2, err := codec.Required[string](req.Parameters, "task")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.Delete(param1, param2)
		if err := call.Context(ctx).Do(); err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.delete", err)
		}
		return workspace.OperationResult{}, nil

	case "tasks.get":
		if err := codec.Validate(req, false, []string{"tasklist", "task"}, []string{}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		param2, err := codec.Required[string](req.Parameters, "task")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.Get(param1, param2)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.get", err)
		}
		return codec.Encode(response)

	case "tasks.insert":
		if err := codec.Validate(req, true, []string{"tasklist"}, []string{"hidden", "parent", "previous"}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		body, err := codec.Body[googletasks.Task](req.Body)
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.Insert(param1, body)
		if _, ok := req.Options["hidden"]; ok {
			value0, err := codec.Optional[bool](req.Options, "hidden")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.Hidden(value0)
		}
		if _, ok := req.Options["parent"]; ok {
			value1, err := codec.Optional[string](req.Options, "parent")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.Parent(value1)
		}
		if _, ok := req.Options["previous"]; ok {
			value2, err := codec.Optional[string](req.Options, "previous")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.Previous(value2)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.insert", err)
		}
		return codec.Encode(response)

	case "tasks.list":
		if err := codec.Validate(req, false, []string{"tasklist"}, []string{"completedMax", "completedMin", "dueMax", "dueMin", "maxResults", "pageToken", "showAssigned", "showCompleted", "showDeleted", "showHidden", "updatedMin"}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.List(param1)
		if _, ok := req.Options["completedMax"]; ok {
			value0, err := codec.Optional[string](req.Options, "completedMax")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.CompletedMax(value0)
		}
		if _, ok := req.Options["completedMin"]; ok {
			value1, err := codec.Optional[string](req.Options, "completedMin")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.CompletedMin(value1)
		}
		if _, ok := req.Options["dueMax"]; ok {
			value2, err := codec.Optional[string](req.Options, "dueMax")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.DueMax(value2)
		}
		if _, ok := req.Options["dueMin"]; ok {
			value3, err := codec.Optional[string](req.Options, "dueMin")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.DueMin(value3)
		}
		if _, ok := req.Options["maxResults"]; ok {
			value4, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.MaxResults(value4)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value5, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.PageToken(value5)
		}
		if _, ok := req.Options["showAssigned"]; ok {
			value6, err := codec.Optional[bool](req.Options, "showAssigned")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.ShowAssigned(value6)
		}
		if _, ok := req.Options["showCompleted"]; ok {
			value7, err := codec.Optional[bool](req.Options, "showCompleted")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.ShowCompleted(value7)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value8, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.ShowDeleted(value8)
		}
		if _, ok := req.Options["showHidden"]; ok {
			value9, err := codec.Optional[bool](req.Options, "showHidden")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.ShowHidden(value9)
		}
		if _, ok := req.Options["updatedMin"]; ok {
			value10, err := codec.Optional[string](req.Options, "updatedMin")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.UpdatedMin(value10)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.list", err)
		}
		return codec.Encode(response)

	case "tasks.move":
		if err := codec.Validate(req, false, []string{"tasklist", "task"}, []string{"destinationTasklist", "parent", "previous"}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		param2, err := codec.Required[string](req.Parameters, "task")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.Move(param1, param2)
		if _, ok := req.Options["destinationTasklist"]; ok {
			value0, err := codec.Optional[string](req.Options, "destinationTasklist")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.DestinationTasklist(value0)
		}
		if _, ok := req.Options["parent"]; ok {
			value1, err := codec.Optional[string](req.Options, "parent")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.Parent(value1)
		}
		if _, ok := req.Options["previous"]; ok {
			value2, err := codec.Optional[string](req.Options, "previous")
			if err != nil {
				return workspace.OperationResult{}, err
			}
			call = call.Previous(value2)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.move", err)
		}
		return codec.Encode(response)

	case "tasks.patch":
		if err := codec.Validate(req, true, []string{"tasklist", "task"}, []string{}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		param2, err := codec.Required[string](req.Parameters, "task")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		body, err := codec.Body[googletasks.Task](req.Body)
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.Patch(param1, param2, body)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.patch", err)
		}
		return codec.Encode(response)

	case "tasks.update":
		if err := codec.Validate(req, true, []string{"tasklist", "task"}, []string{}); err != nil {
			return workspace.OperationResult{}, err
		}
		param1, err := codec.Required[string](req.Parameters, "tasklist")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		param2, err := codec.Required[string](req.Parameters, "task")
		if err != nil {
			return workspace.OperationResult{}, err
		}
		body, err := codec.Body[googletasks.Task](req.Body)
		if err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Tasks.Update(param1, param2, body)
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("tasks", "tasks.update", err)
		}
		return codec.Encode(response)

	default:
		return workspace.OperationResult{}, codec.UnsupportedOperation()
	}
}
