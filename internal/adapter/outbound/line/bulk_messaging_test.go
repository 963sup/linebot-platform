package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
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
	request := messagingdelivery.NarrowcastRequest{Payload: map[string]any{"messages": []any{map[string]any{"type": "text", "text": "hello"}}, "recipient": map[string]any{"type": "operator", "and": []any{}}}}
	if err := sender.SendNarrowcast(context.Background(), request, retryKey); err != nil {
		t.Fatal(err)
	}
	if len(urls) != 3 || urls[0] != "/v2/bot/message/broadcast" || urls[1] != "/v2/bot/message/multicast" || urls[2] != "/v2/bot/message/narrowcast" {
		t.Fatalf("unexpected LINE API paths: %v", urls)
	}
}
