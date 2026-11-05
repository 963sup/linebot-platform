package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
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
		{"list", func() error { _, e := s.ListSurfaces(ctx); return e }},
		{"get", func() error {
			_, e := s.GetSurface(ctx, experiencepresentation.ProviderSurfaceID("richmenu-1"))
			return e
		}},
		{"default", func() error { _, e := s.GetDefaultSurface(ctx); return e }},
		{"alias-list", func() error { _, e := s.ListAliases(ctx); return e }},
		{"alias", func() error { _, e := s.GetAlias(ctx, experiencepresentation.AliasID("alias-1")); return e }},
		{"user", func() error { _, e := s.GetRecipientSurface(ctx, experiencepresentation.RecipientID("U123")); return e }},
		{"progress", func() error { _, e := s.GetBatchProgress(ctx, "request-1"); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
