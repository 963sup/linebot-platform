package line

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCampaignUsesNativeSDKEdgesAndRetryKey(t *testing.T) {
	const retryKey = "123e4567-e89b-42d3-a456-426614174000"
	var urls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer private-token" || r.Header.Get("X-Line-Retry-Key") != retryKey {
			t.Error("invalid campaign request metadata")
		}
		urls = append(urls, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	sender := contractSender(t, server)
	if err := sender.Broadcast(context.Background(), testText(), retryKey); err != nil {
		t.Fatal(err)
	}
	if err := sender.Multicast(context.Background(), []string{"U123"}, testText(), retryKey); err != nil {
		t.Fatal(err)
	}
	request := json.RawMessage(`{"messages":[{"type":"text","text":"hello"}],"recipient":{"type":"operator","and":[]}}`)
	if err := sender.Narrowcast(context.Background(), request, retryKey); err != nil {
		t.Fatal(err)
	}
	if len(urls) != 3 || urls[0] != "/v2/bot/message/broadcast" || urls[1] != "/v2/bot/message/multicast" || urls[2] != "/v2/bot/message/narrowcast" {
		t.Fatalf("unexpected LINE API paths: %v", urls)
	}
}
