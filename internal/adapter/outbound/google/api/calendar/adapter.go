package calendar

import (
    "context"
    "encoding/json"
    "errors"
    "strings"

    "github.com/963sup/linebot-platform/internal/application/ports/outbound"
    googlecalendar "google.golang.org/api/calendar/v3"
)

type Adapter struct {client *googlecalendar.Service}

func New(client *googlecalendar.Service)(*Adapter,error){
    if client==nil{return nil,errors.New("calendar SDK service is required")}
    return &Adapter{client:client},nil
}

func(a *Adapter) ExecuteCalendar(ctx context.Context,req outbound.WorkspaceRequest)(json.RawMessage,error){
    if ctx==nil{return nil,errors.New("workspace context required")}
    if err:=ctx.Err();err!=nil{return nil,err}
    switch strings.SplitN(req.Operation,".",2)[0]{
    case "acl": return a.executeAcl(ctx,req)
    case "calendarList": return a.executeCalendarList(ctx,req)
    case "calendars": return a.executeCalendars(ctx,req)
    case "channels": return a.executeChannels(ctx,req)
    case "colors": return a.executeColors(ctx,req)
    case "events": return a.executeEvents(ctx,req)
    case "freebusy": return a.executeFreebusy(ctx,req)
    case "settings": return a.executeSettings(ctx,req)
    default:return nil,errors.New("unsupported calendar SDK operation")
    }
}
