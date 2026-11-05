package line

import (
	"context"
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
	ingestor, _ := messagingdelivery.NewIngestor(store, queue)
	h := NewDurableHandler(secret, destination, ingestor, nil)
	body := fmt.Sprintf(`{"destination":"Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","events":[{"type":"message","mode":"active","timestamp":%d,"webhookEventId":"01H00ABCDEF0123456789ABCDE","deliveryContext":{"isRedelivery":false},"source":{"type":"user","userId":"Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},"replyToken":"reply-token","message":{"id":"1234567890123","type":"text","text":"hello"}}]}`, time.Now().UnixMilli())
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
		if record.Message.Text == "hello" {
			t.Fatal("incoming user message persisted")
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
	ingestor, _ := messagingdelivery.NewIngestor(store, queue)
	handler := NewDurableHandler(secret, destination, ingestor, nil)
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
