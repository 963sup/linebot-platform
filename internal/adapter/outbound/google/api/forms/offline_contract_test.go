package forms

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	googleforms "google.golang.org/api/forms/v1"
	"google.golang.org/api/option"
)

func TestFormsPinnedSDKRoutes(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		if r.URL.Path == "/v1/forms" && r.Method == http.MethodPost && r.URL.Query().Get("unpublished") != "true" {
			t.Error("unpublished parameter missing on form creation")
		}
		if r.URL.Path == "/v1/forms/form-1/responses" && (r.URL.Query().Get("pageSize") != "5" || r.URL.Query().Get("pageToken") != "next") {
			t.Error("form response pagination options lost")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	svc, err := googleforms.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := New(svc)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		operation  string
		want       string
		parameters map[string]any
		options    map[string]any
		body       json.RawMessage
	}{
		{operation: "forms.batchUpdate", want: "POST /v1/forms/form-1:batchUpdate", parameters: map[string]any{"formId": json.RawMessage("\"form-1\"")}, body: json.RawMessage(`{}`), options: nil},
		{operation: "forms.create", want: "POST /v1/forms", parameters: map[string]any{}, body: json.RawMessage(`{}`), options: map[string]any{"unpublished": json.RawMessage("true")}},
		{operation: "forms.get", want: "GET /v1/forms/form-1", parameters: map[string]any{"formId": json.RawMessage("\"form-1\"")}, body: nil, options: nil},
		{operation: "forms.setPublishSettings", want: "POST /v1/forms/form-1:setPublishSettings", parameters: map[string]any{"formId": json.RawMessage("\"form-1\"")}, body: json.RawMessage(`{}`), options: nil},
		{operation: "forms.responses.get", want: "GET /v1/forms/form-1/responses/response-1", parameters: map[string]any{"formId": json.RawMessage("\"form-1\""), "responseId": json.RawMessage("\"response-1\"")}, body: nil, options: nil},
		{operation: "forms.responses.list", want: "GET /v1/forms/form-1/responses", parameters: map[string]any{"formId": json.RawMessage("\"form-1\"")}, body: nil, options: map[string]any{"pageSize": json.RawMessage("5"), "pageToken": json.RawMessage("\"next\"")}},
		{operation: "forms.watches.create", want: "POST /v1/forms/form-1/watches", parameters: map[string]any{"formId": json.RawMessage("\"form-1\"")}, body: json.RawMessage(`{}`), options: nil},
		{operation: "forms.watches.delete", want: "DELETE /v1/forms/form-1/watches/watch-1", parameters: map[string]any{"formId": json.RawMessage("\"form-1\""), "watchId": json.RawMessage("\"watch-1\"")}, body: nil, options: nil},
		{operation: "forms.watches.list", want: "GET /v1/forms/form-1/watches", parameters: map[string]any{"formId": json.RawMessage("\"form-1\"")}, body: nil, options: nil},
		{operation: "forms.watches.renew", want: "POST /v1/forms/form-1/watches/watch-1:renew", parameters: map[string]any{"formId": json.RawMessage("\"form-1\""), "watchId": json.RawMessage("\"watch-1\"")}, body: json.RawMessage(`{}`), options: nil},
	}
	for _, tc := range cases {
		t.Run(tc.operation, func(t *testing.T) {
			got = ""
			req := workspace.Command{Operation: tc.operation, Parameters: tc.parameters, Options: tc.options, Body: tc.body}
			if _, err := adapter.ExecuteForms(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %s; want %s", got, tc.want)
			}
		})
	}
	if _, err := adapter.ExecuteForms(context.Background(), workspace.Command{Operation: "forms.responses.get"}); err == nil {
		t.Fatal("missing required parameter accepted")
	}
	if _, err := adapter.ExecuteForms(context.Background(), workspace.Command{Operation: "forms.not-a-method"}); err == nil {
		t.Fatal("unsupported endpoint accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.ExecuteForms(ctx, workspace.Command{Operation: "forms.get"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}

func TestFormsRemoteErrorHasNoResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"sensitive-user-info","code":403}}`))
	}))
	defer server.Close()
	svc, err := googleforms.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(svc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.ExecuteForms(context.Background(), workspace.Command{Operation: "forms.get", Parameters: map[string]any{"formId": json.RawMessage(`"form-1"`)}})
	var remote *codec.RemoteError
	if !errors.As(err, &remote) || remote.Status != 403 || remote.API != "forms" {
		t.Fatalf("expected redacted Google 403, got %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "sensitive-user-info") {
		t.Fatal("provider response leaked")
	}
}
