package line

import (
	"context"
	"fmt"
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
		_, _ = w.Write([]byte(`{"audienceGroup":{"audienceGroupId":42,"description":"active audience","status":"READY","created":1700000000}}`))
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

func TestAudienceDetailUnwrapsEnvelopeAndUsesUnixSeconds(t *testing.T) {
	for _, shared := range []bool{false, true} {
		for _, tc := range []struct {
			name, body string
			invalid    bool
		}{
			{"nested", `{"audienceGroup":{"audienceGroupId":42,"description":"active","status":"READY","created":1700000000}}`, false},
			{"missing", `{}`, true}, {"null", `{"audienceGroup":null}`, true},
		} {
			t.Run(fmt.Sprintf("%s/shared=%t", tc.name, shared), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tc.body))
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
				got, err := reader.GetAudience(context.Background(), 42, shared)
				if tc.invalid {
					if err == nil {
						t.Fatal("missing group accepted")
					}
					return
				}
				if err != nil || got.ID != 42 || got.Description != "active" || got.Status != "READY" || got.Shared != shared || got.CreatedAt.Unix() != 1700000000 {
					t.Fatalf("audience=%#v err=%v", got, err)
				}
			})
		}
	}
}

func TestAudienceFlatCreateShapesRemainSupported(t *testing.T) {
	for _, value := range []any{
		&manage_audience.CreateAudienceGroupResponse{AudienceGroupId: 42, Description: "created", Created: 1700000000},
		&manage_audience.CreateClickBasedAudienceGroupResponse{AudienceGroupId: 42, Description: "created", Created: 1700000000},
		&manage_audience.CreateImpBasedAudienceGroupResponse{AudienceGroupId: 42, Description: "created", Created: 1700000000},
	} {
		got, err := mapProviderAudience(value, false)
		if err != nil || got.ID != 42 || got.Description != "created" || got.CreatedAt.Unix() != 1700000000 {
			t.Fatalf("create %T = %#v err=%v", value, got, err)
		}
	}
}
