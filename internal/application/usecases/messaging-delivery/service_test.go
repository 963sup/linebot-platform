package messagingdelivery

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

func testDeliveryContext() DeliveryContext {
	return DeliveryContext{
		TenantID: "tenant-a", ConnectionID: "line-main", ConversationKind: ConversationDirect,
		ConversationReferenceSHA256: strings.Repeat("a", 64), ActorSHA256: strings.Repeat("a", 64),
	}
}

type fakeStore struct {
	recorded   map[string]DeliveryWorkItem
	state      map[string]DeliveryState
	lease      map[string]string
	finished   DeliveryState
	finishedAt time.Time
	pending    []string
}

var (
	_ EventRecorder         = (*fakeStore)(nil)
	_ DeliveryClaimer       = (*fakeStore)(nil)
	_ DeliveryRecovery      = (*fakeStore)(nil)
	_ DeliveryWorkflowStore = (*fakeStore)(nil)
)

type inFlightStore struct{ *fakeStore }

func (f inFlightStore) Claim(context.Context, string, string, time.Time) (DeliveryWorkItem, bool, error) {
	return DeliveryWorkItem{}, false, ErrDeliveryInFlight
}

func newFakeStore() *fakeStore {
	return &fakeStore{recorded: map[string]DeliveryWorkItem{}, state: map[string]DeliveryState{}, lease: map[string]string{}}
}

func (f *fakeStore) Record(_ context.Context, job DeliveryWorkItem) (bool, error) {
	if previous, exists := f.recorded[job.Key]; exists {
		if (f.state[job.Key] == valueobjects.StatusPending || f.state[job.Key] == valueobjects.StatusExpired) && job.ReplyUntil.After(previous.ReplyUntil) {
			previous.ReplyUntil = job.ReplyUntil
			previous.ReplyToken = job.ReplyToken
			f.recorded[job.Key] = previous
			f.state[job.Key] = valueobjects.StatusPending
		}
		return false, nil
	}
	f.recorded[job.Key] = job
	f.state[job.Key] = valueobjects.StatusPending
	return true, nil
}
func (f *fakeStore) Claim(_ context.Context, key, lease string, now time.Time) (DeliveryWorkItem, bool, error) {
	job, ok := f.recorded[key]
	if !ok {
		return DeliveryWorkItem{}, false, errors.New("missing job")
	}
	if f.state[key] != valueobjects.StatusPending {
		return DeliveryWorkItem{}, false, nil
	}
	if !now.Before(job.ReplyUntil) {
		f.state[key] = valueobjects.StatusExpired
		return DeliveryWorkItem{}, false, nil
	}
	f.state[key] = valueobjects.StatusSending
	f.lease[key] = lease
	return job, true, nil
}
func (f *fakeStore) Complete(_ context.Context, key, lease string, outcome DeliveryState, completedAt time.Time) error {
	if f.state[key] != valueobjects.StatusSending || f.lease[key] != lease {
		return errors.New("invalid lease")
	}
	f.state[key] = outcome
	f.finished = outcome
	f.finishedAt = completedAt
	return nil
}
func (f *fakeStore) Recover(_ context.Context, _ time.Time, _ int) ([]string, error) {
	return f.pending, nil
}

type fakeQueue struct {
	keys []EventKey
	err  error
}

func (q *fakeQueue) Enqueue(_ context.Context, key EventKey) error {
	q.keys = append(q.keys, key)
	return q.err
}

type fakeSender struct {
	calls   int
	err     error
	onReply func()
}

func (f *fakeSender) Reply(_ context.Context, _ string, _ []Message) error {
	f.calls++
	if f.onReply != nil {
		f.onReply()
	}
	return f.err
}
func (f *fakeSender) Push(_ context.Context, _ string, _ []Message, _ string) error {
	return errors.New("not used")
}

func TestEventKeyIsStableAndPrivate(t *testing.T) {
	key, err := NewEventKey("channel", "event-id")
	if err != nil {
		t.Fatalf("invalid event key: %v", err)
	}
	same, err := NewEventKey("channel", "event-id")
	if err != nil || same != key {
		t.Fatal("event key is not stable")
	}
	other, _ := NewEventKey("other", "event-id")
	if key == other {
		t.Fatal("channel isolation failed")
	}
	for _, key := range []string{"", "bad", "A000000000000000000000000000000000000000000000000000000000000000"} {
		if _, err := ParseEventKey(key); err == nil {
			t.Fatal("accepted invalid event key")
		}
	}
}

func TestDeliveryContextRejectsMissingOrInvalidScope(t *testing.T) {
	valid := testDeliveryContext()
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.ActorSHA256 = strings.Repeat("b", 64)
	if err := invalid.Validate(); err == nil {
		t.Fatal("direct conversation accepted a different actor")
	}
	invalid = valid
	invalid.ConversationReferenceSHA256 = "provider-user-id"
	if err := invalid.Validate(); err == nil {
		t.Fatal("raw provider reference accepted")
	}
}

func TestProcessorRejectsZeroEventKeyBeforeStoreAccess(t *testing.T) {
	store := newFakeStore()
	processor, err := NewProcessor(store, &fakeSender{})
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), EventKey{}); err == nil {
		t.Fatal("zero EventKey must be rejected")
	}
	if len(store.recorded) != 0 {
		t.Fatal("invalid key reached the event store")
	}
}

func TestProcessorPreservesInFlightRetrySignal(t *testing.T) {
	processor, err := NewProcessor(inFlightStore{newFakeStore()}, &fakeSender{})
	if err != nil {
		t.Fatal(err)
	}
	key, _ := ParseEventKey(strings.Repeat("a", 64))
	if err := processor.Process(context.Background(), key); !errors.Is(err, ErrDeliveryInFlight) {
		t.Fatalf("expected in-flight retry signal, got %v", err)
	}
}

func TestDuplicateReceiptDoesNotOverwriteAndRequeues(t *testing.T) {
	store, queue := newFakeStore(), &fakeQueue{}
	i, err := NewIngestor(store, queue)
	if err != nil {
		t.Fatal(err)
	}
	i.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	msg := Message{Kind: KindText, Text: "fixed response"}
	if err := i.Ingest(context.Background(), testDeliveryContext(), "event", "reply-one", i.now(), msg); err != nil {
		t.Fatal(err)
	}
	if err := i.Ingest(context.Background(), testDeliveryContext(), "event", "reply-two", i.now(), msg); err != nil {
		t.Fatal(err)
	}
	if len(store.recorded) != 1 || len(queue.keys) != 2 {
		t.Fatal("duplicate receipt must persist once and re-enqueue safely")
	}
	key := queue.keys[0]
	if queue.keys[1] != key || store.recorded[key.String()].ReplyToken != "reply-one" {
		t.Fatal("duplicate overwrote original payload or generated a new key")
	}
}

func TestProcessorSendsAtMostOnce(t *testing.T) {
	store, queue, sender := newFakeStore(), &fakeQueue{}, &fakeSender{}
	i, _ := NewIngestor(store, queue)
	now := time.Now().UTC()
	i.now = func() time.Time { return now }
	msg := Message{Kind: KindText, Text: "fixed response"}
	if err := i.Ingest(context.Background(), testDeliveryContext(), "event", "reply", now, msg); err != nil {
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
	if sender.calls != 1 || store.finished != DeliverySent {
		t.Fatalf("calls=%d finish=%s", sender.calls, store.finished)
	}
	if !store.finishedAt.Equal(now) {
		t.Fatalf("completion time=%s, want %s", store.finishedAt, now)
	}
}

func TestProcessorRecordsCompletionAfterReplyReturns(t *testing.T) {
	store, queue := newFakeStore(), &fakeQueue{}
	base := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	ingestor, _ := NewIngestor(store, queue)
	ingestor.now = func() time.Time { return base }
	if err := ingestor.Ingest(context.Background(), testDeliveryContext(), "timed-event", "reply", base, Message{Kind: KindText, Text: "fixed"}); err != nil {
		t.Fatal(err)
	}

	clockValues := []time.Time{base, base.Add(time.Second), base.Add(2 * time.Second)}
	clockCalls := 0
	sender := &fakeSender{onReply: func() {
		if clockCalls != 2 {
			t.Fatalf("Reply called after %d clock reads, want claim and attempt reads", clockCalls)
		}
	}}
	processor, _ := NewProcessor(store, sender)
	processor.now = func() time.Time {
		if clockCalls >= len(clockValues) {
			t.Fatalf("unexpected clock read %d", clockCalls+1)
		}
		value := clockValues[clockCalls]
		clockCalls++
		return value
	}

	if err := processor.Process(context.Background(), queue.keys[0]); err != nil {
		t.Fatal(err)
	}
	if clockCalls != 3 {
		t.Fatalf("clock reads=%d, want claim, attempt, completion", clockCalls)
	}
	if !store.finishedAt.Equal(base.Add(2 * time.Second)) {
		t.Fatalf("completion time=%s, want post-reply time", store.finishedAt)
	}
}

func TestUnknownOutcomeDoesNotResend(t *testing.T) {
	store, queue, sender := newFakeStore(), &fakeQueue{}, &fakeSender{err: errors.New("network timeout")}
	i, _ := NewIngestor(store, queue)
	now := time.Now()
	i.now = func() time.Time { return now }
	if err := i.Ingest(context.Background(), testDeliveryContext(), "event", "reply", now, Message{Kind: KindText, Text: "fixed"}); err != nil {
		t.Fatal(err)
	}
	p, _ := NewProcessor(store, sender)
	p.now = func() time.Time { return now }
	for j := 0; j < 2; j++ {
		if err := p.Process(context.Background(), queue.keys[0]); err != nil {
			t.Fatal(err)
		}
	}
	if sender.calls != 1 || store.finished != DeliveryUnknown {
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

func TestDispatcherRejectsInvalidPersistedKey(t *testing.T) {
	store, queue := newFakeStore(), &fakeQueue{}
	store.pending = []string{"bad"}
	dispatcher, err := NewDispatcher(store, queue)
	if err != nil {
		t.Fatal(err)
	}
	if err := dispatcher.Dispatch(context.Background()); err == nil {
		t.Fatal("expected invalid persisted key to be rejected")
	}
	if len(queue.keys) != 0 {
		t.Fatal("invalid persisted key reached queue adapter")
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
	message := Message{Kind: KindText, Text: "fixed"}
	if err := ingestor.Ingest(context.Background(), testDeliveryContext(), "redelivery", "token", first, message); err != nil {
		t.Fatal(err)
	}
	key := queue.keys[0]
	current = first.Add(55 * time.Second)
	if err := processor.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 0 || store.state[key.String()] != "expired" {
		t.Fatal("expired reply was sent")
	}
	if err := ingestor.Ingest(context.Background(), testDeliveryContext(), "redelivery", "token", first, message); err != nil {
		t.Fatal(err)
	}
	if store.state[key.String()] != "pending" {
		t.Fatal("signed redelivery should restore never-sent expired work")
	}
	if err := processor.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if err := ingestor.Ingest(context.Background(), testDeliveryContext(), "redelivery", "token", first, message); err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 || store.state[key.String()] != "sent" {
		t.Fatalf("a redelivery resent an already used reply: calls=%d state=%q", sender.calls, store.state[key.String()])
	}
}

func TestRedeliveryWindowUsesOriginalTimestamp(t *testing.T) {
	store, queue := newFakeStore(), &fakeQueue{}
	ingestor, _ := NewIngestor(store, queue)
	now := time.Date(2026, 11, 3, 0, 20, 1, 0, time.UTC)
	ingestor.now = func() time.Time { return now }
	message := Message{Kind: KindText, Text: "fixed"}
	old := now.Add(-20*time.Minute - time.Second)
	if err := ingestor.Ingest(context.Background(), testDeliveryContext(), "too-old", "token", old, message); !errors.Is(err, ErrEventTooOld) {
		t.Fatalf("expected ErrEventTooOld, got %v", err)
	}
	if len(store.recorded) != 0 || len(queue.keys) != 0 {
		t.Fatal("late redelivery was accepted for send")
	}
	if err := ingestor.Ingest(context.Background(), testDeliveryContext(), "near-limit", "token", now.Add(-19*time.Minute-50*time.Second), message); err != nil {
		t.Fatal(err)
	}
	for _, job := range store.recorded {
		if job.ReplyUntil.After(job.OccurredAt.Add(20 * time.Minute)) {
			t.Fatal("reply cutoff exceeds original event age")
		}
	}
}

type failFirstEnqueue struct{ calls int }

func (q *failFirstEnqueue) Enqueue(_ context.Context, _ EventKey) error {
	q.calls++
	if q.calls == 1 {
		return errors.New("transient queue failure")
	}
	return nil
}

func TestDispatcherAttemptsAllReservedKeysOnQueueError(t *testing.T) {
	store := newFakeStore()
	store.pending = []string{
		"0000000000000000000000000000000000000000000000000000000000000001",
		"0000000000000000000000000000000000000000000000000000000000000002",
	}
	queue := &failFirstEnqueue{}
	dispatcher, err := NewDispatcher(store, queue)
	if err != nil {
		t.Fatal(err)
	}
	if err := dispatcher.Dispatch(context.Background()); err == nil {
		t.Fatal("expected partial queue failure to be reported")
	}
	if queue.calls != len(store.pending) {
		t.Fatalf("later reserved tasks were skipped, calls=%d", queue.calls)
	}
}
