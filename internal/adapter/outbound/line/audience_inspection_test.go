package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
)

func TestAudienceReadOnlySDKEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("invalid audience request")
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "list") {
			_, _ = w.Write([]byte(`{"audienceGroups":[]}`))
			return
		}
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
		{"group", func() error { _, e := s.GetAudience(ctx, 1, false); return e }},
		{"shared", func() error { _, e := s.GetAudience(ctx, 1, true); return e }},
		{"list", func() error { _, e := s.ListAudiences(ctx, audience.PageRequest{Number: 1, Size: 10}, false); return e }},
		{"sharedlist", func() error { _, e := s.ListAudiences(ctx, audience.PageRequest{Number: 1, Size: 10}, true); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAudienceListRejectsMissingGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	client, err := manage_audience.NewManageAudienceAPI("private-token", manage_audience.WithEndpoint(server.URL), manage_audience.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewAudienceReader(client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ListAudiences(context.Background(), audience.PageRequest{Number: 1, Size: 10}, false); err == nil {
		t.Fatal("audience page without audienceGroups accepted")
	}
}

func TestAudienceListMapsTypedPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"audienceGroups":[{"audienceGroupId":42,"description":"active audience","status":"READY"}],"hasNextPage":true,"page":3}`))
	}))
	defer server.Close()
	client, err := manage_audience.NewManageAudienceAPI("private-token", manage_audience.WithEndpoint(server.URL), manage_audience.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewAudienceReader(client)
	if err != nil {
		t.Fatal(err)
	}
	page, err := reader.ListAudiences(context.Background(), audience.PageRequest{Number: 3, Size: 10}, false)
	if err != nil {
		t.Fatal(err)
	}
	if page.Page != 3 || page.NextPage != 4 || len(page.Audiences) != 1 || page.Audiences[0].ID != 42 || page.Audiences[0].Description != "active audience" {
		t.Fatalf("mapped page = %#v", page)
	}
}
