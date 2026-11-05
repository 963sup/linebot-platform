package identity

import (
	"context"
	"errors"
	"testing"
	"time"
)

type outboxFake struct {
	items       []ReservedIntegrationEvent
	resolutions []DispatchResolution
}

func (f *outboxFake) ReserveIdentityEvents(context.Context, time.Time, int) ([]ReservedIntegrationEvent, error) {
	return append([]ReservedIntegrationEvent(nil), f.items...), nil
}
func (f *outboxFake) ResolveIdentityEvent(_ context.Context, _, _ string, result DispatchResolution, _ time.Time) error {
	f.resolutions = append(f.resolutions, result)
	return nil
}

type publisherFake struct {
	outcomes []PublishOutcome
	errs     []error
	calls    int
}

func (f *publisherFake) PublishIdentityEvent(context.Context, IntegrationEvent) (PublishOutcome, error) {
	index := f.calls
	f.calls++
	return f.outcomes[index], f.errs[index]
}

func TestIdentityDispatcherClassifiesPublishResults(t *testing.T) {
	outbox := &outboxFake{items: []ReservedIntegrationEvent{
		{Event: IntegrationEvent{EventID: "confirmed"}, LeaseID: "l1"},
		{Event: IntegrationEvent{EventID: "rejected"}, LeaseID: "l2"},
		{Event: IntegrationEvent{EventID: "unknown"}, LeaseID: "l3"},
	}}
	publisher := &publisherFake{
		outcomes: []PublishOutcome{PublishConfirmed, PublishDefinitelyNotSent, PublishUnknown},
		errs:     []error{nil, errors.New("rejected"), context.DeadlineExceeded},
	}
	dispatcher, err := NewIdentityEventDispatcher(outbox, publisher)
	if err != nil {
		t.Fatal(err)
	}
	if err := dispatcher.Dispatch(context.Background()); err == nil {
		t.Fatal("publish failures were not reported")
	}
	want := []DispatchResolution{DispatchPublished, DispatchRetryable, DispatchUnknown}
	if len(outbox.resolutions) != len(want) {
		t.Fatalf("resolutions = %v", outbox.resolutions)
	}
	for index := range want {
		if outbox.resolutions[index] != want[index] {
			t.Fatalf("resolution[%d] = %v, want %v", index, outbox.resolutions[index], want[index])
		}
	}
}

func TestIdentityDispatcherTreatsUnclassifiedErrorAsUnknown(t *testing.T) {
	outbox := &outboxFake{items: []ReservedIntegrationEvent{{Event: IntegrationEvent{EventID: "event"}, LeaseID: "lease"}}}
	publisher := &publisherFake{outcomes: []PublishOutcome{0}, errs: []error{errors.New("transport failed")}}
	dispatcher, _ := NewIdentityEventDispatcher(outbox, publisher)
	if err := dispatcher.Dispatch(context.Background()); err == nil {
		t.Fatal("invalid outcome accepted")
	}
	if len(outbox.resolutions) != 1 || outbox.resolutions[0] != DispatchUnknown {
		t.Fatalf("resolutions = %v, want unknown", outbox.resolutions)
	}
}

func TestIdentityDispatcherRejectsCanceledContextBeforeReserve(t *testing.T) {
	dispatcher, _ := NewIdentityEventDispatcher(&outboxFake{}, &publisherFake{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(dispatcher.Dispatch(ctx), context.Canceled) {
		t.Fatal("canceled context not preserved")
	}
}

type durableOutboxFake struct {
	item       ReservedIntegrationEvent
	state      string
	resolution DispatchResolution
}

func (f *durableOutboxFake) ReserveIdentityEvents(context.Context, time.Time, int) ([]ReservedIntegrationEvent, error) {
	if f.state != "pending" {
		return nil, nil
	}
	f.state = "publishing"
	return []ReservedIntegrationEvent{f.item}, nil
}

func (f *durableOutboxFake) ResolveIdentityEvent(_ context.Context, _, _ string, result DispatchResolution, _ time.Time) error {
	f.resolution = result
	switch result {
	case DispatchPublished:
		f.state = "published"
	case DispatchRetryable:
		f.state = "pending"
	case DispatchUnknown:
		f.state = "unknown"
	}
	return nil
}

func TestIdentityDispatcherRestartDoesNotRepublishTerminalEvent(t *testing.T) {
	store := &durableOutboxFake{
		state: "pending",
		item: ReservedIntegrationEvent{Event: IntegrationEvent{
			EventID: "event", IdempotencyKey: "event",
		}, LeaseID: "lease"},
	}
	publisher := &publisherFake{outcomes: []PublishOutcome{PublishConfirmed}, errs: []error{nil}}
	first, _ := NewIdentityEventDispatcher(store, publisher)
	if err := first.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	restarted, _ := NewIdentityEventDispatcher(store, publisher)
	if err := restarted.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if publisher.calls != 1 || store.state != "published" {
		t.Fatalf("calls=%d state=%q, want one published delivery", publisher.calls, store.state)
	}
}

func TestIdentityDispatcherUnknownOutcomeIsNotRepublished(t *testing.T) {
	store := &durableOutboxFake{
		state: "pending",
		item:  ReservedIntegrationEvent{Event: IntegrationEvent{EventID: "event"}, LeaseID: "lease"},
	}
	publisher := &publisherFake{outcomes: []PublishOutcome{PublishUnknown}, errs: []error{context.DeadlineExceeded}}
	dispatcher, _ := NewIdentityEventDispatcher(store, publisher)
	if err := dispatcher.Dispatch(context.Background()); err == nil {
		t.Fatal("unknown provider result not reported")
	}
	if err := dispatcher.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if publisher.calls != 1 || store.state != "unknown" {
		t.Fatalf("calls=%d state=%q, want one unknown delivery", publisher.calls, store.state)
	}
}
