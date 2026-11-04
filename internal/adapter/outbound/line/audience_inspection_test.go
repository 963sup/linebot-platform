package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
)

func TestAudienceReadOnlySDKEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("invalid audience request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	client, err := manage_audience.NewManageAudienceAPI("private-token", manage_audience.WithEndpoint(server.URL), manage_audience.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewAudienceReader(client)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"group", func() error { _, e := s.AudienceGroup(ctx, 1); return e }},
		{"shared", func() error { _, e := s.SharedAudienceGroup(ctx, 1); return e }},
		{"list", func() error { _, e := s.AudienceGroups(ctx, 1, 10); return e }},
		{"sharedlist", func() error { _, e := s.SharedAudienceGroups(ctx, 1, 10); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
