package line

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

type persisted struct {
	seen map[string]messagingdelivery.DeliveryWorkItem
}

func (f *persisted) Record(_ context.Context, job messagingdelivery.DeliveryWorkItem) (bool, error) {
	if _, ok := f.seen[job.Key]; ok {
		return false, nil
	}
	f.seen[job.Key] = job
	return true, nil
}
func (f *persisted) Claim(context.Context, string, string, time.Time) (messagingdelivery.DeliveryWorkItem, bool, error) {
	return messagingdelivery.DeliveryWorkItem{}, false, nil
}
func (f *persisted) Complete(context.Context, string, string, messagingdelivery.DeliveryState, time.Time) error {
	return nil
}
func (f *persisted) Recover(context.Context, time.Time, int) ([]string, error) { return nil, nil }

type workQueue struct {
	keys []messagingdelivery.EventKey
	fail bool
}

func (q *workQueue) Enqueue(_ context.Context, key messagingdelivery.EventKey) error {
	if q.fail {
		return errors.New("network failure")
	}
	q.keys = append(q.keys, key)
	return nil
}

func TestDurableWebhookSignatureDestinationDedupAndEnqueueFailure(t *testing.T) {
	secret := "test-secret"
	destination := "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	store := &persisted{seen: map[string]messagingdelivery.DeliveryWorkItem{}}
	queue := &workQueue{}
	ingestor, _ := messagingdelivery.NewIngestor(store, queue, destination)
	policy, _ := messagingdelivery.NewReplyPolicy("https://miniapp.line.me/2000000000-example")
	h := NewDurableHandler(secret, destination, ingestor, policy, nil)
	body := fmt.Sprintf(`{"destination":"Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","events":[{"type":"message","mode":"active","timestamp":%d,"webhookEventId":"01H00ABCDEF0123456789ABCDE","deliveryContext":{"isRedelivery":false},"source":{"type":"group","groupId":"Cxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","userId":"Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},"replyToken":"reply-token","message":{"id":"1234567890123","type":"text","text":"MINI App"}}]}`, time.Now().UnixMilli())
	for n := 0; n < 2; n++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, signedRequest(secret, body))
		if w.Code != http.StatusOK {
			t.Fatalf("valid signed webhook: %d", w.Code)
		}
	}
	if len(store.seen) != 1 || len(queue.keys) != 2 {
		t.Fatalf("durable dedup failed: inbox=%d dispatched=%d", len(store.seen), len(queue.keys))
	}
	for _, record := range store.seen {
		if record.Message.Kind != messagingdelivery.KindFlex || record.Message.Flex.Action == nil || record.Message.Flex.Action.URI != "https://miniapp.line.me/2000000000-example" {
			t.Fatalf("unexpected persisted reply: %#v", record.Message)
		}
		if record.Context.ConversationKind != messagingdelivery.ConversationGroup ||
			record.Context.ConversationReferenceSHA256 == "Cxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" ||
			record.Context.ActorSHA256 == "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" {
			t.Fatalf("unsafe or missing conversation scope: %#v", record.Context)
		}
	}
	queue.fail = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, signedRequest(secret, body))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("enqueue failure should return 503, got %d", w.Code)
	}
	bad := httptest.NewRecorder()
	h.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/webhook/line", strings.NewReader(body)))
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature status: %d", bad.Code)
	}
}

func TestDurableWebhookIgnoresStandbyAndRejectsMissingDeliveryContext(t *testing.T) {
	secret := "test-secret"
	destination := "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	store := &persisted{seen: map[string]messagingdelivery.DeliveryWorkItem{}}
	queue := &workQueue{}
	ingestor, _ := messagingdelivery.NewIngestor(store, queue, destination)
	policy, _ := messagingdelivery.NewReplyPolicy("")
	handler := NewDurableHandler(secret, destination, ingestor, policy, nil)
	base := fmt.Sprintf(`{"destination":"%s","events":[{"type":"message","mode":"%%s","timestamp":%d,"webhookEventId":"01H00ABCDEF0123456789ABCDE",%%s"source":{"type":"user","userId":"%s"},"replyToken":"reply-token","message":{"id":"1234567890123","type":"text","text":"hello"}}]}`, destination, time.Now().UnixMilli(), destination)

	standby := fmt.Sprintf(base, "standby", `"deliveryContext":{"isRedelivery":false},`)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, signedRequest(secret, standby))
	if w.Code != http.StatusOK || len(queue.keys) != 0 {
		t.Fatalf("standby event status=%d queued=%d", w.Code, len(queue.keys))
	}

	missing := fmt.Sprintf(base, "active", "")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, signedRequest(secret, missing))
	if w.Code != http.StatusBadRequest || len(queue.keys) != 0 {
		t.Fatalf("missing delivery context status=%d queued=%d", w.Code, len(queue.keys))
	}
}

func signedRequest(secret, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/webhook/line", strings.NewReader(body))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	r.Header.Set("X-Line-Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestDurableOnlyIngressGuardsAndUnsupportedRoom(t *testing.T) {
	const secret, destination = "test-secret", "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	for _, tc := range []struct {
		name    string
		request *http.Request
		status  int
	}{
		{"method", httptest.NewRequest(http.MethodGet, "/webhook/line", nil), http.StatusMethodNotAllowed},
		{"oversize", signedRequest(secret, strings.Repeat("x", maxBodyBytes+1)), http.StatusRequestEntityTooLarge},
		{"malformed", signedRequest(secret, "{"), http.StatusBadRequest},
		{"destination mismatch", signedRequest(secret, `{"destination":"other","events":[]}`), http.StatusUnauthorized},
		{"empty verification", signedRequest(secret, `{"destination":"`+destination+`","events":[]}`), http.StatusOK},
		{"room ignored", signedRequest(secret, `{"destination":"`+destination+`","events":[{"type":"message","mode":"active","source":{"type":"room","roomId":"R-room"},"replyToken":"token","message":{"type":"text","id":"1","text":"hello"}}]}`), http.StatusOK},
		{"unsupported event", signedRequest(secret, `{"destination":"`+destination+`","events":[{"type":"join","source":{"type":"group","groupId":"C-group"}}]}`), http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &persisted{seen: map[string]messagingdelivery.DeliveryWorkItem{}}
			queue := &workQueue{}
			ingestor, _ := messagingdelivery.NewIngestor(store, queue, destination)
			policy, _ := messagingdelivery.NewReplyPolicy("")
			handler := NewDurableHandler(secret, destination, ingestor, policy, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.request)
			if w.Code != tc.status || len(store.seen) != 0 || len(queue.keys) != 0 {
				t.Fatalf("status=%d want=%d records=%d queued=%d", w.Code, tc.status, len(store.seen), len(queue.keys))
			}
		})
	}
}
