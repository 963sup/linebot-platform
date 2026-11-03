package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeStore struct {
	recorded map[string]outbound.EventJob
	state    map[string]string
	lease    map[string]string
	finished outbound.DeliveryState
	pending  []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{recorded: map[string]outbound.EventJob{}, state: map[string]string{}, lease: map[string]string{}}
}

func (f *fakeStore) Record(_ context.Context, job outbound.EventJob) (bool, error) {
	if previous, exists := f.recorded[job.Key]; exists {
		if (f.state[job.Key] == "pending" || f.state[job.Key] == "expired") && job.ReplyUntil.After(previous.ReplyUntil) {
			previous.ReplyUntil = job.ReplyUntil
			previous.ReplyToken = job.ReplyToken
			f.recorded[job.Key] = previous
			f.state[job.Key] = "pending"
		}
		return false, nil
	}
	f.recorded[job.Key] = job
	f.state[job.Key] = "pending"
	return true, nil
}
func (f *fakeStore) Claim(_ context.Context, key, lease string, now time.Time) (outbound.EventJob, bool, error) {
	job, ok := f.recorded[key]
	if !ok {
		return outbound.EventJob{}, false, errors.New("missing job")
	}
	if f.state[key] != "pending" {
		return outbound.EventJob{}, false, nil
	}
	if !now.Before(job.ReplyUntil) {
		f.state[key] = "expired"
		return outbound.EventJob{}, false, nil
	}
	f.state[key] = "sending"
	f.lease[key] = lease
	return job, true, nil
}
func (f *fakeStore) Complete(_ context.Context, key, lease string, outcome outbound.DeliveryState) error {
	if f.state[key] != "sending" || f.lease[key] != lease {
		return errors.New("invalid lease")
	}
	f.state[key] = string(outcome)
	f.finished = outcome
	return nil
}
func (f *fakeStore) Recover(_ context.Context, _ time.Time, _ int) ([]string, error) {
	return f.pending, nil
}

type fakeQueue struct {
	keys []string
	err  error
}

func (q *fakeQueue) Enqueue(_ context.Context, key string) error {
	q.keys = append(q.keys, key)
	return q.err
}

type fakeSender struct {
	calls int
	err   error
}

func (f *fakeSender) Reply(_ context.Context, _ string, _ []outbound.Message) error {
	f.calls++
	return f.err
}
func (f *fakeSender) Push(_ context.Context, _ string, _ []outbound.Message, _ string) error {
	return errors.New("not used")
}

func TestEventKeyIsStableAndPrivate(t *testing.T) {
	key, err := EventKey("channel", "event-id")
	if err != nil || !ValidKey(key) {
		t.Fatalf("invalid event key: %v", err)
	}
	same, err := EventKey("channel", "event-id")
	if err != nil || same != key {
		t.Fatal("event key is not stable")
	}
	other, _ := EventKey("other", "event-id")
	if key == other {
		t.Fatal("channel isolation failed")
	}
	for _, key := range []string{"", "bad", "A000000000000000000000000000000000000000000000000000000000000000"} {
		if ValidKey(key) {
			t.Fatal("accepted invalid event key")
		}
	}
}

func TestDuplicateReceiptDoesNotOverwriteAndRequeues(t *testing.T) {
	store, queue := newFakeStore(), &fakeQueue{}
	i, err := NewIngestor(store, queue)
	if err != nil {
		t.Fatal(err)
	}
	i.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	msg := outbound.Message{Kind: outbound.KindText, Text: "fixed response"}
	if err := i.Ingest(context.Background(), "channel", "event", "reply-one", i.now(), msg); err != nil {
		t.Fatal(err)
	}
	if err := i.Ingest(context.Background(), "channel", "event", "reply-two", i.now(), msg); err != nil {
		t.Fatal(err)
	}
	if len(store.recorded) != 1 || len(queue.keys) != 2 {
		t.Fatal("duplicate receipt must persist once and re-enqueue safely")
	}
	key := queue.keys[0]
	if queue.keys[1] != key || store.recorded[key].ReplyToken != "reply-one" {
		t.Fatal("duplicate overwrote original payload or generated a new key")
	}
}

func TestProcessorSendsAtMostOnce(t *testing.T) {
	store, queue, sender := newFakeStore(), &fakeQueue{}, &fakeSender{}
	i, _ := NewIngestor(store, queue)
	now := time.Now().UTC()
	i.now = func() time.Time { return now }
	msg := outbound.Message{Kind: outbound.KindText, Text: "fixed response"}
	if err := i.Ingest(context.Background(), "channel", "event", "reply", now, msg); err != nil {
		t.Fatal(err)
	}
	p, _ := NewProcessor(store, sender)
	p.now = func() time.Time { return now }
	key := queue.keys[0]
	if err := p.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if err := p.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 || store.finished != outbound.DeliverySent {
		t.Fatalf("calls=%d finish=%s", sender.calls, store.finished)
	}
}

func TestUnknownOutcomeDoesNotResend(t *testing.T) {
	store, queue, sender := newFakeStore(), &fakeQueue{}, &fakeSender{err: errors.New("network timeout")}
	i, _ := NewIngestor(store, queue)
	now := time.Now()
	i.now = func() time.Time { return now }
	if err := i.Ingest(context.Background(), "channel", "event", "reply", now, outbound.Message{Kind: outbound.KindText, Text: "fixed"}); err != nil {
		t.Fatal(err)
	}
	p, _ := NewProcessor(store, sender)
	p.now = func() time.Time { return now }
	for j := 0; j < 2; j++ {
		if err := p.Process(context.Background(), queue.keys[0]); err != nil {
			t.Fatal(err)
		}
	}
	if sender.calls != 1 || store.finished != outbound.DeliveryUnknown {
		t.Fatalf("unsafe retry: calls=%d outcome=%s", sender.calls, store.finished)
	}
}

func TestDispatcherReEnqueuesPending(t *testing.T) {
	store, queue := newFakeStore(), &fakeQueue{}
	store.pending = []string{"0000000000000000000000000000000000000000000000000000000000000000"}
	d, err := NewDispatcher(store, queue)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(queue.keys) != 1 {
		t.Fatal("dispatcher did not enqueue pending event")
	}
	queue.err = errors.New("transient enqueue error")
	if err := d.Dispatch(context.Background()); err == nil {
		t.Fatal("expected retryable enqueue error")
	}
}

// TestRedeliveryRefreshesOnlyNeverSentWork guards LINE's one-use reply token.
func TestRedeliveryRefreshesOnlyNeverSentWork(t *testing.T) {
	store, queue, sender := newFakeStore(), &fakeQueue{}, &fakeSender{}
	ingestor, _ := NewIngestor(store, queue)
	processor, _ := NewProcessor(store, sender)
	first := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	current := first
	ingestor.now = func() time.Time { return current }
	processor.now = func() time.Time { return current }
	message := outbound.Message{Kind: outbound.KindText, Text: "fixed"}
	if err := ingestor.Ingest(context.Background(), "channel", "redelivery", "token", first, message); err != nil {
		t.Fatal(err)
	}
	key := queue.keys[0]
	current = first.Add(55 * time.Second)
	if err := processor.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 0 || store.state[key] != "expired" {
		t.Fatal("expired reply was sent")
	}
	if err := ingestor.Ingest(context.Background(), "channel", "redelivery", "token", first, message); err != nil {
		t.Fatal(err)
	}
	if store.state[key] != "pending" {
		t.Fatal("signed redelivery should restore never-sent expired work")
	}
	if err := processor.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if err := ingestor.Ingest(context.Background(), "channel", "redelivery", "token", first, message); err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 || store.state[key] != "sent" {
		t.Fatalf("a redelivery resent an already used reply: calls=%d state=%q", sender.calls, store.state[key])
	}
}

func TestRedeliveryWindowUsesOriginalTimestamp(t *testing.T) {
	store, queue := newFakeStore(), &fakeQueue{}
	ingestor, _ := NewIngestor(store, queue)
	now := time.Date(2026, 11, 3, 0, 20, 1, 0, time.UTC)
	ingestor.now = func() time.Time { return now }
	message := outbound.Message{Kind: outbound.KindText, Text: "fixed"}
	old := now.Add(-20*time.Minute - time.Second)
	if err := ingestor.Ingest(context.Background(), "channel", "too-old", "token", old, message); !errors.Is(err, ErrEventTooOld) {
		t.Fatalf("expected ErrEventTooOld, got %v", err)
	}
	if len(store.recorded) != 0 || len(queue.keys) != 0 {
		t.Fatal("late redelivery was accepted for send")
	}
	if err := ingestor.Ingest(context.Background(), "channel", "near-limit", "token", now.Add(-19*time.Minute-50*time.Second), message); err != nil {
		t.Fatal(err)
	}
	for _, job := range store.recorded {
		if job.ReplyUntil.After(job.OccurredAt.Add(20 * time.Minute)) {
			t.Fatal("reply cutoff exceeds original event age")
		}
	}
}
