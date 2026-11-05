package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookConfigurationUsesOfficialSDK(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("missing token")
		}
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v2/bot/channel/webhook/test" {
			_, _ = w.Write([]byte(`{"success":true,"statusCode":200,"reason":"OK"}`))
			return
		}
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	if err := s.SetEndpoint(context.Background(), "https://example.com/webhook"); err != nil {
		t.Fatal(err)
	}
	r, err := s.TestEndpoint(context.Background(), "https://example.com/webhook")
	if err != nil || !r.Reachable || r.StatusCode != 200 {
		t.Fatalf("unexpected test result: %+v %v", r, err)
	}
	if len(paths) != 2 || paths[0] != "PUT /v2/bot/channel/webhook/endpoint" || paths[1] != "POST /v2/bot/channel/webhook/test" {
		t.Fatalf("wrong paths: %v", paths)
	}
}
