package calendar

import (
	"context"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
)

func (a *Adapter) executeColors(ctx context.Context, req workspace.Command) (workspace.Result, error) {
	switch req.Operation {
	case "colors.get":
		if err := codec.Validate(req, false, []string{}, []string{}); err != nil {
			return workspace.Result{}, err
		}
		call := a.client.Colors.Get()
		response, err := call.Context(ctx).Do()
		if err != nil {
			return workspace.Result{}, codec.Wrap("calendar", "colors.get", err)
		}
		return codec.Encode(response)

	default:
		return workspace.Result{}, codec.UnsupportedOperation()
	}
}
