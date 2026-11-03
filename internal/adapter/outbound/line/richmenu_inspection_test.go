package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRichMenuReadsUseNativeSDK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"list", func() error { _, e := s.RichMenuList(ctx); return e }},
		{"get", func() error { _, e := s.RichMenu(ctx, "richmenu-1"); return e }},
		{"default", func() error { _, e := s.DefaultRichMenu(ctx); return e }},
		{"alias-list", func() error { _, e := s.RichMenuAliasList(ctx); return e }},
		{"alias", func() error { _, e := s.RichMenuAlias(ctx, "alias-1"); return e }},
		{"user", func() error { _, e := s.UserRichMenu(ctx, "U123"); return e }},
		{"progress", func() error { _, e := s.RichMenuBatchProgress(ctx, "request-1"); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
