package calendar

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
)

func (a *Adapter) executeColors(ctx context.Context, req codec.Request) (workspace.OperationResult, error) {
	switch req.Operation {
	case "colors.get":
		if err := codec.Validate(req, false, []string{}, []string{}); err != nil {
			return workspace.OperationResult{}, err
		}
		call := a.client.Colors.Get()
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.OperationResult{}, codec.Wrap("calendar", "colors.get", err)
		}
		return codec.Encode(response)

	default:
		return workspace.OperationResult{}, codec.UnsupportedOperation()
	}
}
