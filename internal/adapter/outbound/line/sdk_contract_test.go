package line

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func contractSender(t *testing.T, server *httptest.Server) *Sender {
	t.Helper()
	client, err := messaging_api.NewMessagingApiAPI("private-token",
		messaging_api.WithEndpoint(server.URL),
		messaging_api.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	sender, err := NewSender(client)
	if err != nil {
		t.Fatal(err)
	}
	return sender
}

func testText() []messagingdelivery.Message {
	return []messagingdelivery.Message{{Kind: messagingdelivery.KindText, Text: "hello"}}
}

func TestReplyAndPushUseNativeLINEAPI(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	var retryKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer private-token" {
			t.Errorf("unexpected authorization header")
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected HTTP method %s", r.Method)
		}
		mu.Lock()
		paths = append(paths, r.URL.Path)
		retryKeys = append(retryKeys, r.Header.Get("X-Line-Retry-Key"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	if err := s.Reply(context.Background(), "reply-token", testText()); err != nil {
		t.Fatal(err)
	}
	key := "123e4567-e89b-42d3-a456-426614174000"
	if err := s.Push(context.Background(), "U00000000000000000000000000000000", testText(), key); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 2 || paths[0] != "/v2/bot/message/reply" || paths[1] != "/v2/bot/message/push" {
		t.Fatalf("unexpected paths %v", paths)
	}
	if retryKeys[0] != "" || retryKeys[1] != key {
		t.Fatalf("unexpected retry key headers %v", retryKeys)
	}
}

func TestLINEAPIErrorIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "private-message-and-token", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	s := contractSender(t, server)
	err := s.Push(context.Background(), "U00000000000000000000000000000000", testText(), "retry-key")
	var external *resilience.ExternalError
	if !errors.As(err, &external) || external.Kind != resilience.KindUnavailable || external.StatusCode != 503 {
		t.Fatalf("unexpected error classification: %v", err)
	}
	if strings.Contains(err.Error(), "private-message-and-token") {
		t.Fatal("SDK response body must not leak through Error()")
	}
	if external.OutcomeUnknown {
		t.Fatal("HTTP 503 response is known, not a missing response")
	}
}

type interceptTransport func(*http.Request) (*http.Response, error)

func (f interceptTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type closeTrackingBody struct{ closed bool }

func (b *closeTrackingBody) Read([]byte) (int, error) { return 0, http.ErrBodyReadAfterClose }
func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

func TestLINEAPIErrorClosesSDKErrorResponse(t *testing.T) {
	body := &closeTrackingBody{}
	err := lineAPIError("reply", &http.Response{StatusCode: http.StatusBadGateway, Body: body}, errors.New("provider detail"))
	if err == nil || !body.closed {
		t.Fatal("LINE error response body must be closed")
	}
	if strings.Contains(err.Error(), "provider detail") {
		t.Fatal("provider error detail must be sanitized")
	}
}

func TestLINEContextCancellation(t *testing.T) {
	started := make(chan struct{})
	transport := interceptTransport(func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	client, err := messaging_api.NewMessagingApiAPI("token",
		messaging_api.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSender(client)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Reply(ctx, "reply-token", testText()) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("SDK request did not reach the configured transport")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation through SDK request context, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SDK request did not honor caller cancellation")
	}
}

func TestLINEParallelRequestsHaveIsolatedContexts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)

	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := s.Reply(ctx, fmt.Sprintf("reply-token-%d", i), testText()); err != nil {
				failures <- err
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
}
