package calendar

import (
    "context"
    "encoding/json"
    "github.com/963sup/linebot-platform/internal/application/ports/outbound"
    "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
)

func (a *Adapter) executeColors(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage,error) {
    switch req.Operation {
case "colors.get":
        if err := codec.Validate(req, false, []string{}, []string{}); err != nil {return nil, err}
        call := a.client.Colors.Get()
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("calendar", "colors.get",err)}
        return codec.Encode(response)

    default:
        return nil, codec.UnsupportedOperation()
    }
}
