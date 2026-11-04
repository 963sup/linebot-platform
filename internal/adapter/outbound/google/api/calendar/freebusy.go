package calendar

import (
    "context"
    "encoding/json"
    "github.com/963sup/linebot-platform/internal/application/ports/outbound"
    "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
    googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeFreebusy(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage,error) {
    switch req.Operation {
case "freebusy.query":
        if err := codec.Validate(req, true, []string{}, []string{}); err != nil {return nil, err}
        body, err := codec.Body[googlecalendar.FreeBusyRequest](req.Body)
        if err != nil {return nil, err}
        call := a.client.Freebusy.Query(body)
        response, err := call.Context(ctx).Do()
        if err != nil {return nil, codec.Wrap("calendar", "freebusy.query",err)}
        return codec.Encode(response)

    default:
        return nil, codec.UnsupportedOperation()
    }
}
