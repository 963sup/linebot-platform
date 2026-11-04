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

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/ingestion"
)

type persisted struct {
	seen map[string]outbound.EventJob
}

func (f *persisted) Record(_ context.Context, job outbound.EventJob) (bool, error) {
	if _, ok := f.seen[job.Key]; ok {
		return false, nil
	}
	f.seen[job.Key] = job
	return true, nil
}
func (f *persisted) Claim(context.Context, string, string, time.Time) (outbound.EventJob, bool, error) {
	return outbound.EventJob{}, false, nil
}
func (f *persisted) Complete(context.Context, string, string, outbound.DeliveryState) error {
	return nil
}
func (f *persisted) Recover(context.Context, time.Time, int) ([]string, error) { return nil, nil }

type workQueue struct {
	keys []string
	fail bool
}

func (q *workQueue) Enqueue(_ context.Context, key string) error {
	if q.fail {
		return errors.New("network failure")
	}
	q.keys = append(q.keys, key)
	return nil
}

func TestDurableWebhookSignatureDestinationDedupAndEnqueueFailure(t *testing.T) {
	secret := "test-secret"
	destination := "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	store := &persisted{seen: map[string]outbound.EventJob{}}
	queue := &workQueue{}
	ingestor, _ := ingestion.NewIngestor(store, queue)
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
