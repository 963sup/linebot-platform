package line

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRichMenuMutationsUseOfficialEndpoints(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("missing channel access token")
		}
		seen = append(seen, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { return s.CreateRichMenu(ctx, json.RawMessage(`{"areas":[]}`)) },
		func() error { return s.ValidateRichMenu(ctx, json.RawMessage(`{"areas":[]}`)) },
		func() error {
			return s.CreateRichMenuAlias(ctx, json.RawMessage(`{"richMenuAliasId":"main","richMenuId":"richmenu-1"}`))
		},
		func() error {
			return s.UpdateRichMenuAlias(ctx, "main", json.RawMessage(`{"richMenuId":"richmenu-2"}`))
		},
		func() error {
			return s.BulkLinkRichMenu(ctx, json.RawMessage(`{"richMenuId":"richmenu-1","userIds":["U1"]}`))
		},
		func() error { return s.BulkUnlinkRichMenu(ctx, json.RawMessage(`{"userIds":["U1"]}`)) },
		func() error { return s.RichMenuBatch(ctx, json.RawMessage(`{"operations":[]}`)) },
		func() error { return s.ValidateRichMenuBatch(ctx, json.RawMessage(`{"operations":[]}`)) },
		func() error { return s.SetDefaultRichMenu(ctx, "richmenu-1") },
		func() error { return s.CancelDefaultRichMenu(ctx) },
		func() error { return s.LinkRichMenuToUser(ctx, "U123", "richmenu-1") },
		func() error { return s.UnlinkRichMenuFromUser(ctx, "U123") },
		func() error { return s.DeleteRichMenu(ctx, "richmenu-1") },
		func() error { return s.DeleteRichMenuAlias(ctx, "alias-1") },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 14 {
		t.Fatalf("expected fourteen SDK calls, got %v", seen)
	}
}
