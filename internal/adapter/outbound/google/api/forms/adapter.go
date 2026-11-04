package forms

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	googleforms "google.golang.org/api/forms/v1"
)

// Adapter executes only statically enumerated Forms v1 operations via the
// official generated SDK. It never owns user OAuth credentials.
type Adapter struct{ client *googleforms.Service }

func New(client *googleforms.Service) (*Adapter, error) {
	if client == nil { return nil, errors.New("Forms SDK service is required") }
	return &Adapter{client: client}, nil
}

func (a *Adapter) ExecuteForms(ctx context.Context, req outbound.WorkspaceRequest) (json.RawMessage, error) {
	if ctx == nil { return nil, errors.New("workspace context required") }
	if err := ctx.Err(); err != nil { return nil, err }
	switch req.Operation {
	case "forms.batchUpdate":
		if err := codec.Validate(req, true, []string{"formId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		body, err := codec.Body[googleforms.BatchUpdateFormRequest](req.Body)
		if err != nil { return nil, err }
		call := a.client.Forms.BatchUpdate(pFormId, body)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.batchUpdate", err) }
		return codec.Encode(result)
	case "forms.create":
		if err := codec.Validate(req, true, []string{}, []string{"unpublished"}); err != nil { return nil, err }
		body, err := codec.Body[googleforms.Form](req.Body)
		if err != nil { return nil, err }
		call := a.client.Forms.Create(body)
		if _, ok := req.Options["unpublished"]; ok {
			value, err := codec.Optional[bool](req.Options, "unpublished")
			if err != nil { return nil, err }
			call = call.Unpublished(value)
		}
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.create", err) }
		return codec.Encode(result)
	case "forms.get":
		if err := codec.Validate(req, false, []string{"formId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		call := a.client.Forms.Get(pFormId)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.get", err) }
		return codec.Encode(result)
	case "forms.setPublishSettings":
		if err := codec.Validate(req, true, []string{"formId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		body, err := codec.Body[googleforms.SetPublishSettingsRequest](req.Body)
		if err != nil { return nil, err }
		call := a.client.Forms.SetPublishSettings(pFormId, body)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.setPublishSettings", err) }
		return codec.Encode(result)
	case "forms.responses.get":
		if err := codec.Validate(req, false, []string{"formId", "responseId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		pResponseId, err := codec.Required[string](req.Parameters, "responseId")
		if err != nil { return nil, err }
		call := a.client.Forms.Responses.Get(pFormId, pResponseId)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.responses.get", err) }
		return codec.Encode(result)
	case "forms.responses.list":
		if err := codec.Validate(req, false, []string{"formId"}, []string{"filter", "pageSize", "pageToken"}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		call := a.client.Forms.Responses.List(pFormId)
		if _, ok := req.Options["filter"]; ok {
			value, err := codec.Optional[string](req.Options, "filter")
			if err != nil { return nil, err }
			call = call.Filter(value)
		}
		if _, ok := req.Options["pageSize"]; ok {
			value, err := codec.Optional[int64](req.Options, "pageSize")
			if err != nil { return nil, err }
			call = call.PageSize(value)
		}
		if _, ok := req.Options["pageToken"]; ok {
			value, err := codec.Optional[string](req.Options, "pageToken")
			if err != nil { return nil, err }
			call = call.PageToken(value)
		}
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.responses.list", err) }
		return codec.Encode(result)
	case "forms.watches.create":
		if err := codec.Validate(req, true, []string{"formId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		body, err := codec.Body[googleforms.CreateWatchRequest](req.Body)
		if err != nil { return nil, err }
		call := a.client.Forms.Watches.Create(pFormId, body)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.watches.create", err) }
		return codec.Encode(result)
	case "forms.watches.delete":
		if err := codec.Validate(req, false, []string{"formId", "watchId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		pWatchId, err := codec.Required[string](req.Parameters, "watchId")
		if err != nil { return nil, err }
		call := a.client.Forms.Watches.Delete(pFormId, pWatchId)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.watches.delete", err) }
		return codec.Encode(result)
	case "forms.watches.list":
		if err := codec.Validate(req, false, []string{"formId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		call := a.client.Forms.Watches.List(pFormId)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.watches.list", err) }
		return codec.Encode(result)
	case "forms.watches.renew":
		if err := codec.Validate(req, true, []string{"formId", "watchId"}, []string{}); err != nil { return nil, err }
		pFormId, err := codec.Required[string](req.Parameters, "formId")
		if err != nil { return nil, err }
		pWatchId, err := codec.Required[string](req.Parameters, "watchId")
		if err != nil { return nil, err }
		body, err := codec.Body[googleforms.RenewWatchRequest](req.Body)
		if err != nil { return nil, err }
		call := a.client.Forms.Watches.Renew(pFormId, pWatchId, body)
		result, err := call.Context(ctx).Do()
		if err != nil { return nil, codec.Wrap("forms", "forms.watches.renew", err) }
		return codec.Encode(result)
	default: return nil, codec.UnsupportedOperation()
	}
}
