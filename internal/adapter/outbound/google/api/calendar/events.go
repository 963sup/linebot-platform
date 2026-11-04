package calendar

import (
	"context"
	"encoding/json"
	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	googlecalendar "google.golang.org/api/calendar/v3"
)

func (a *Adapter) executeEvents(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage, error) {
	switch req.Operation {
	case "events.delete":
		if err := codec.Validate(req, false, []string{"calendarId", "eventId"}, []string{"sendNotifications", "sendUpdates"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "eventId")
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Delete(param1, param2)
		if _, ok := req.Options["sendNotifications"]; ok {
			value0, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value0)
		}
		if _, ok := req.Options["sendUpdates"]; ok {
			value1, err := codec.Optional[string](req.Options, "sendUpdates")
			if err != nil {
				return nil, err
			}
			call = call.SendUpdates(value1)
		}
		if err := call.Context(ctx).Do(); err != nil {
			return nil, codec.Wrap("calendar", "events.delete", err)
		}
		return nil, nil

	case "events.get":
		if err := codec.Validate(req, false, []string{"calendarId", "eventId"}, []string{"alwaysIncludeEmail", "maxAttendees", "timeZone"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "eventId")
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Get(param1, param2)
		if _, ok := req.Options["alwaysIncludeEmail"]; ok {
			value0, err := codec.Optional[bool](req.Options, "alwaysIncludeEmail")
			if err != nil {
				return nil, err
			}
			call = call.AlwaysIncludeEmail(value0)
		}
		if _, ok := req.Options["maxAttendees"]; ok {
			value1, err := codec.Optional[int64](req.Options, "maxAttendees")
			if err != nil {
				return nil, err
			}
			call = call.MaxAttendees(value1)
		}
		if _, ok := req.Options["timeZone"]; ok {
			value2, err := codec.Optional[string](req.Options, "timeZone")
			if err != nil {
				return nil, err
			}
			call = call.TimeZone(value2)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.get", err)
		}
		return codec.Encode(response)

	case "events.import":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{"conferenceDataVersion", "eventLabelVersion", "supportsAttachments"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.Event](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Import(param1, body)
		if _, ok := req.Options["conferenceDataVersion"]; ok {
			value0, err := codec.Optional[int64](req.Options, "conferenceDataVersion")
			if err != nil {
				return nil, err
			}
			call = call.ConferenceDataVersion(value0)
		}
		if _, ok := req.Options["eventLabelVersion"]; ok {
			value1, err := codec.Optional[int64](req.Options, "eventLabelVersion")
			if err != nil {
				return nil, err
			}
			call = call.EventLabelVersion(value1)
		}
		if _, ok := req.Options["supportsAttachments"]; ok {
			value2, err := codec.Optional[bool](req.Options, "supportsAttachments")
			if err != nil {
				return nil, err
			}
			call = call.SupportsAttachments(value2)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.import", err)
		}
		return codec.Encode(response)

	case "events.insert":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{"conferenceDataVersion", "eventLabelVersion", "maxAttendees", "sendNotifications", "sendUpdates", "supportsAttachments"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.Event](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Insert(param1, body)
		if _, ok := req.Options["conferenceDataVersion"]; ok {
			value0, err := codec.Optional[int64](req.Options, "conferenceDataVersion")
			if err != nil {
				return nil, err
			}
			call = call.ConferenceDataVersion(value0)
		}
		if _, ok := req.Options["eventLabelVersion"]; ok {
			value1, err := codec.Optional[int64](req.Options, "eventLabelVersion")
			if err != nil {
				return nil, err
			}
			call = call.EventLabelVersion(value1)
		}
		if _, ok := req.Options["maxAttendees"]; ok {
			value2, err := codec.Optional[int64](req.Options, "maxAttendees")
			if err != nil {
				return nil, err
			}
			call = call.MaxAttendees(value2)
		}
		if _, ok := req.Options["sendNotifications"]; ok {
			value3, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value3)
		}
		if _, ok := req.Options["sendUpdates"]; ok {
			value4, err := codec.Optional[string](req.Options, "sendUpdates")
			if err != nil {
				return nil, err
			}
			call = call.SendUpdates(value4)
		}
		if _, ok := req.Options["supportsAttachments"]; ok {
			value5, err := codec.Optional[bool](req.Options, "supportsAttachments")
			if err != nil {
				return nil, err
			}
			call = call.SupportsAttachments(value5)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.insert", err)
		}
		return codec.Encode(response)

	case "events.instances":
		if err := codec.Validate(req, false, []string{"calendarId", "eventId"}, []string{"alwaysIncludeEmail", "maxAttendees", "maxResults", "originalStart", "pageToken", "showDeleted", "timeMax", "timeMin", "timeZone"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "eventId")
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Instances(param1, param2)
		if _, ok := req.Options["alwaysIncludeEmail"]; ok {
			value0, err := codec.Optional[bool](req.Options, "alwaysIncludeEmail")
			if err != nil {
				return nil, err
			}
			call = call.AlwaysIncludeEmail(value0)
		}
		if _, ok := req.Options["maxAttendees"]; ok {
			value1, err := codec.Optional[int64](req.Options, "maxAttendees")
			if err != nil {
				return nil, err
			}
			call = call.MaxAttendees(value1)
		}
		if _, ok := req.Options["maxResults"]; ok {
			value2, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return nil, err
			}
			call = call.MaxResults(value2)
		}
		if _, ok := req.Options["originalStart"]; ok {
			value3, err := codec.Optional[string](req.Options, "originalStart")
			if err != nil {
				return nil, err
			}
			call = call.OriginalStart(value3)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value4, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return nil, err
			}
			call = call.PageToken(value4)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value5, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return nil, err
			}
			call = call.ShowDeleted(value5)
		}
		if _, ok := req.Options["timeMax"]; ok {
			value6, err := codec.Optional[string](req.Options, "timeMax")
			if err != nil {
				return nil, err
			}
			call = call.TimeMax(value6)
		}
		if _, ok := req.Options["timeMin"]; ok {
			value7, err := codec.Optional[string](req.Options, "timeMin")
			if err != nil {
				return nil, err
			}
			call = call.TimeMin(value7)
		}
		if _, ok := req.Options["timeZone"]; ok {
			value8, err := codec.Optional[string](req.Options, "timeZone")
			if err != nil {
				return nil, err
			}
			call = call.TimeZone(value8)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.instances", err)
		}
		return codec.Encode(response)

	case "events.list":
		if err := codec.Validate(req, false, []string{"calendarId"}, []string{"alwaysIncludeEmail", "eventTypes", "iCalUID", "maxAttendees", "maxResults", "orderBy", "pageToken", "privateExtendedProperty", "q", "sharedExtendedProperty", "showDeleted", "showHiddenInvitations", "singleEvents", "syncToken", "timeMax", "timeMin", "timeZone", "updatedMin"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		call := a.client.Events.List(param1)
		if _, ok := req.Options["alwaysIncludeEmail"]; ok {
			value0, err := codec.Optional[bool](req.Options, "alwaysIncludeEmail")
			if err != nil {
				return nil, err
			}
			call = call.AlwaysIncludeEmail(value0)
		}
		if _, ok := req.Options["eventTypes"]; ok {
			value1, err := codec.Optional[[]string](req.Options, "eventTypes")
			if err != nil {
				return nil, err
			}
			call = call.EventTypes(value1...)
		}
		if _, ok := req.Options["iCalUID"]; ok {
			value2, err := codec.Optional[string](req.Options, "iCalUID")
			if err != nil {
				return nil, err
			}
			call = call.ICalUID(value2)
		}
		if _, ok := req.Options["maxAttendees"]; ok {
			value3, err := codec.Optional[int64](req.Options, "maxAttendees")
			if err != nil {
				return nil, err
			}
			call = call.MaxAttendees(value3)
		}
		if _, ok := req.Options["maxResults"]; ok {
			value4, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return nil, err
			}
			call = call.MaxResults(value4)
		}
		if _, ok := req.Options["orderBy"]; ok {
			value5, err := codec.Optional[string](req.Options, "orderBy")
			if err != nil {
				return nil, err
			}
			call = call.OrderBy(value5)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value6, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return nil, err
			}
			call = call.PageToken(value6)
		}
		if _, ok := req.Options["privateExtendedProperty"]; ok {
			value7, err := codec.Optional[[]string](req.Options, "privateExtendedProperty")
			if err != nil {
				return nil, err
			}
			call = call.PrivateExtendedProperty(value7...)
		}
		if _, ok := req.Options["q"]; ok {
			value8, err := codec.Optional[string](req.Options, "q")
			if err != nil {
				return nil, err
			}
			call = call.Q(value8)
		}
		if _, ok := req.Options["sharedExtendedProperty"]; ok {
			value9, err := codec.Optional[[]string](req.Options, "sharedExtendedProperty")
			if err != nil {
				return nil, err
			}
			call = call.SharedExtendedProperty(value9...)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value10, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return nil, err
			}
			call = call.ShowDeleted(value10)
		}
		if _, ok := req.Options["showHiddenInvitations"]; ok {
			value11, err := codec.Optional[bool](req.Options, "showHiddenInvitations")
			if err != nil {
				return nil, err
			}
			call = call.ShowHiddenInvitations(value11)
		}
		if _, ok := req.Options["singleEvents"]; ok {
			value12, err := codec.Optional[bool](req.Options, "singleEvents")
			if err != nil {
				return nil, err
			}
			call = call.SingleEvents(value12)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value13, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return nil, err
			}
			call = call.SyncToken(value13)
		}
		if _, ok := req.Options["timeMax"]; ok {
			value14, err := codec.Optional[string](req.Options, "timeMax")
			if err != nil {
				return nil, err
			}
			call = call.TimeMax(value14)
		}
		if _, ok := req.Options["timeMin"]; ok {
			value15, err := codec.Optional[string](req.Options, "timeMin")
			if err != nil {
				return nil, err
			}
			call = call.TimeMin(value15)
		}
		if _, ok := req.Options["timeZone"]; ok {
			value16, err := codec.Optional[string](req.Options, "timeZone")
			if err != nil {
				return nil, err
			}
			call = call.TimeZone(value16)
		}
		if _, ok := req.Options["updatedMin"]; ok {
			value17, err := codec.Optional[string](req.Options, "updatedMin")
			if err != nil {
				return nil, err
			}
			call = call.UpdatedMin(value17)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.list", err)
		}
		return codec.Encode(response)

	case "events.move":
		if err := codec.Validate(req, false, []string{"calendarId", "eventId", "destination"}, []string{"sendNotifications", "sendUpdates"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "eventId")
		if err != nil {
			return nil, err
		}
		param3, err := codec.Required[string](req.Parameters, "destination")
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Move(param1, param2, param3)
		if _, ok := req.Options["sendNotifications"]; ok {
			value0, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value0)
		}
		if _, ok := req.Options["sendUpdates"]; ok {
			value1, err := codec.Optional[string](req.Options, "sendUpdates")
			if err != nil {
				return nil, err
			}
			call = call.SendUpdates(value1)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.move", err)
		}
		return codec.Encode(response)

	case "events.patch":
		if err := codec.Validate(req, true, []string{"calendarId", "eventId"}, []string{"alwaysIncludeEmail", "conferenceDataVersion", "eventLabelVersion", "maxAttendees", "sendNotifications", "sendUpdates", "supportsAttachments"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "eventId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.Event](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Patch(param1, param2, body)
		if _, ok := req.Options["alwaysIncludeEmail"]; ok {
			value0, err := codec.Optional[bool](req.Options, "alwaysIncludeEmail")
			if err != nil {
				return nil, err
			}
			call = call.AlwaysIncludeEmail(value0)
		}
		if _, ok := req.Options["conferenceDataVersion"]; ok {
			value1, err := codec.Optional[int64](req.Options, "conferenceDataVersion")
			if err != nil {
				return nil, err
			}
			call = call.ConferenceDataVersion(value1)
		}
		if _, ok := req.Options["eventLabelVersion"]; ok {
			value2, err := codec.Optional[int64](req.Options, "eventLabelVersion")
			if err != nil {
				return nil, err
			}
			call = call.EventLabelVersion(value2)
		}
		if _, ok := req.Options["maxAttendees"]; ok {
			value3, err := codec.Optional[int64](req.Options, "maxAttendees")
			if err != nil {
				return nil, err
			}
			call = call.MaxAttendees(value3)
		}
		if _, ok := req.Options["sendNotifications"]; ok {
			value4, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value4)
		}
		if _, ok := req.Options["sendUpdates"]; ok {
			value5, err := codec.Optional[string](req.Options, "sendUpdates")
			if err != nil {
				return nil, err
			}
			call = call.SendUpdates(value5)
		}
		if _, ok := req.Options["supportsAttachments"]; ok {
			value6, err := codec.Optional[bool](req.Options, "supportsAttachments")
			if err != nil {
				return nil, err
			}
			call = call.SupportsAttachments(value6)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.patch", err)
		}
		return codec.Encode(response)

	case "events.quickAdd":
		if err := codec.Validate(req, false, []string{"calendarId", "text"}, []string{"sendNotifications", "sendUpdates"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "text")
		if err != nil {
			return nil, err
		}
		call := a.client.Events.QuickAdd(param1, param2)
		if _, ok := req.Options["sendNotifications"]; ok {
			value0, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value0)
		}
		if _, ok := req.Options["sendUpdates"]; ok {
			value1, err := codec.Optional[string](req.Options, "sendUpdates")
			if err != nil {
				return nil, err
			}
			call = call.SendUpdates(value1)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.quickAdd", err)
		}
		return codec.Encode(response)

	case "events.update":
		if err := codec.Validate(req, true, []string{"calendarId", "eventId"}, []string{"alwaysIncludeEmail", "conferenceDataVersion", "eventLabelVersion", "maxAttendees", "sendNotifications", "sendUpdates", "supportsAttachments"}); err != nil {
			return nil, err
		}
		param1, err := codec.Required[string](req.Parameters, "calendarId")
		if err != nil {
			return nil, err
		}
		param2, err := codec.Required[string](req.Parameters, "eventId")
		if err != nil {
			return nil, err
		}
		body, err := codec.Body[googlecalendar.Event](req.Body)
		if err != nil {
			return nil, err
		}
		call := a.client.Events.Update(param1, param2, body)
		if _, ok := req.Options["alwaysIncludeEmail"]; ok {
			value0, err := codec.Optional[bool](req.Options, "alwaysIncludeEmail")
			if err != nil {
				return nil, err
			}
			call = call.AlwaysIncludeEmail(value0)
		}
		if _, ok := req.Options["conferenceDataVersion"]; ok {
			value1, err := codec.Optional[int64](req.Options, "conferenceDataVersion")
			if err != nil {
				return nil, err
			}
			call = call.ConferenceDataVersion(value1)
		}
		if _, ok := req.Options["eventLabelVersion"]; ok {
			value2, err := codec.Optional[int64](req.Options, "eventLabelVersion")
			if err != nil {
				return nil, err
			}
			call = call.EventLabelVersion(value2)
		}
		if _, ok := req.Options["maxAttendees"]; ok {
			value3, err := codec.Optional[int64](req.Options, "maxAttendees")
			if err != nil {
				return nil, err
			}
			call = call.MaxAttendees(value3)
		}
		if _, ok := req.Options["sendNotifications"]; ok {
			value4, err := codec.Optional[bool](req.Options, "sendNotifications")
			if err != nil {
				return nil, err
			}
			call = call.SendNotifications(value4)
		}
		if _, ok := req.Options["sendUpdates"]; ok {
			value5, err := codec.Optional[string](req.Options, "sendUpdates")
			if err != nil {
				return nil, err
			}
			call = call.SendUpdates(value5)
		}
		if _, ok := req.Options["supportsAttachments"]; ok {
			value6, err := codec.Optional[bool](req.Options, "supportsAttachments")
			if err != nil {
				return nil, err
			}
			call = call.SupportsAttachments(value6)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.update", err)
		}
		return codec.Encode(response)

	case "events.watch":
		if err := codec.Validate(req, true, []string{"calendarId"}, []string{"alwaysIncludeEmail", "eventTypes", "iCalUID", "maxAttendees", "maxResults", "orderBy", "pageToken", "privateExtendedProperty", "q", "sharedExtendedProperty", "showDeleted", "showHiddenInvitations", "singleEvents", "syncToken", "timeMax", "timeMin", "timeZone", "updatedMin"}); err != nil {
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
		call := a.client.Events.Watch(param1, body)
		if _, ok := req.Options["alwaysIncludeEmail"]; ok {
			value0, err := codec.Optional[bool](req.Options, "alwaysIncludeEmail")
			if err != nil {
				return nil, err
			}
			call = call.AlwaysIncludeEmail(value0)
		}
		if _, ok := req.Options["eventTypes"]; ok {
			value1, err := codec.Optional[[]string](req.Options, "eventTypes")
			if err != nil {
				return nil, err
			}
			call = call.EventTypes(value1...)
		}
		if _, ok := req.Options["iCalUID"]; ok {
			value2, err := codec.Optional[string](req.Options, "iCalUID")
			if err != nil {
				return nil, err
			}
			call = call.ICalUID(value2)
		}
		if _, ok := req.Options["maxAttendees"]; ok {
			value3, err := codec.Optional[int64](req.Options, "maxAttendees")
			if err != nil {
				return nil, err
			}
			call = call.MaxAttendees(value3)
		}
		if _, ok := req.Options["maxResults"]; ok {
			value4, err := codec.Optional[int64](req.Options, "maxResults")
			if err != nil {
				return nil, err
			}
			call = call.MaxResults(value4)
		}
		if _, ok := req.Options["orderBy"]; ok {
			value5, err := codec.Optional[string](req.Options, "orderBy")
			if err != nil {
				return nil, err
			}
			call = call.OrderBy(value5)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value6, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil {
				return nil, err
			}
			call = call.PageToken(value6)
		}
		if _, ok := req.Options["privateExtendedProperty"]; ok {
			value7, err := codec.Optional[[]string](req.Options, "privateExtendedProperty")
			if err != nil {
				return nil, err
			}
			call = call.PrivateExtendedProperty(value7...)
		}
		if _, ok := req.Options["q"]; ok {
			value8, err := codec.Optional[string](req.Options, "q")
			if err != nil {
				return nil, err
			}
			call = call.Q(value8)
		}
		if _, ok := req.Options["sharedExtendedProperty"]; ok {
			value9, err := codec.Optional[[]string](req.Options, "sharedExtendedProperty")
			if err != nil {
				return nil, err
			}
			call = call.SharedExtendedProperty(value9...)
		}
		if _, ok := req.Options["showDeleted"]; ok {
			value10, err := codec.Optional[bool](req.Options, "showDeleted")
			if err != nil {
				return nil, err
			}
			call = call.ShowDeleted(value10)
		}
		if _, ok := req.Options["showHiddenInvitations"]; ok {
			value11, err := codec.Optional[bool](req.Options, "showHiddenInvitations")
			if err != nil {
				return nil, err
			}
			call = call.ShowHiddenInvitations(value11)
		}
		if _, ok := req.Options["singleEvents"]; ok {
			value12, err := codec.Optional[bool](req.Options, "singleEvents")
			if err != nil {
				return nil, err
			}
			call = call.SingleEvents(value12)
		}
		if _, ok := req.Options["syncToken"]; ok {
			value13, err := codec.Optional[string](req.Options, "syncToken")
			if err != nil {
				return nil, err
			}
			call = call.SyncToken(value13)
		}
		if _, ok := req.Options["timeMax"]; ok {
			value14, err := codec.Optional[string](req.Options, "timeMax")
			if err != nil {
				return nil, err
			}
			call = call.TimeMax(value14)
		}
		if _, ok := req.Options["timeMin"]; ok {
			value15, err := codec.Optional[string](req.Options, "timeMin")
			if err != nil {
				return nil, err
			}
			call = call.TimeMin(value15)
		}
		if _, ok := req.Options["timeZone"]; ok {
			value16, err := codec.Optional[string](req.Options, "timeZone")
			if err != nil {
				return nil, err
			}
			call = call.TimeZone(value16)
		}
		if _, ok := req.Options["updatedMin"]; ok {
			value17, err := codec.Optional[string](req.Options, "updatedMin")
			if err != nil {
				return nil, err
			}
			call = call.UpdatedMin(value17)
		}
		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, codec.Wrap("calendar", "events.watch", err)
		}
		return codec.Encode(response)

	default:
		return nil, codec.UnsupportedOperation()
	}
}
