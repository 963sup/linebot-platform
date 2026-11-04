package line

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
)

func TestAudienceManagementUsesOfficialEndpoints(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("missing audience bearer")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	client, err := manage_audience.NewManageAudienceAPI("private-token", manage_audience.WithEndpoint(srv.URL), manage_audience.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	blob, err := manage_audience.NewManageAudienceBlobAPI("private-token", manage_audience.WithBlobEndpoint(srv.URL), manage_audience.WithBlobHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewAudienceManager(client, blob)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	j := json.RawMessage("{}")
	tests := []struct {
		want string
		call func() error
	}{
		{"POST /v2/bot/audienceGroup/upload", func() error { _, e := m.CreateAudienceGroup(ctx, j); return e }},
		{"PUT /v2/bot/audienceGroup/upload", func() error { return m.AddAudienceToGroup(ctx, j) }},
		{"POST /v2/bot/audienceGroup/click", func() error { _, e := m.CreateClickAudience(ctx, j); return e }},
		{"POST /v2/bot/audienceGroup/imp", func() error { _, e := m.CreateImpressionAudience(ctx, j); return e }},
		{"PUT /v2/bot/audienceGroup/3/updateDescription", func() error { return m.UpdateAudienceDescription(ctx, 3, j) }},
		{"DELETE /v2/bot/audienceGroup/3", func() error { return m.DeleteAudienceGroup(ctx, 3) }},
		{"POST /v2/bot/audienceGroup/upload/byFile", func() error { _, e := m.CreateAudienceFromFile(ctx, []byte("U123"), "test", false, "upload"); return e }},
		{"PUT /v2/bot/audienceGroup/upload/byFile", func() error { return m.AddAudienceFromFile(ctx, 3, []byte("U123"), "upload") }},
	}
	for _, tt := range tests {
		got = ""
		if err := tt.call(); err != nil {
			t.Fatalf("%s: %v", tt.want, err)
		}
		if got != tt.want {
			t.Fatalf("got %s want %s", got, tt.want)
		}
	}
}
