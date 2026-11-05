package identity

import (
	"context"
	"errors"
	"time"
)

const identityDispatchBatch = 100

// IntegrationEvent is the provider-neutral envelope committed atomically with
// an AccountLink. It intentionally contains no OAuth credential material.
type IntegrationEvent struct {
	EventID          string
	EventType        string
	SchemaVersion    int
	OccurredAt       time.Time
	AggregateID      string
	AggregateVersion int64
	CorrelationID    string
	IdempotencyKey   string
}

// ReservedIntegrationEvent carries the opaque lease required to complete a
// dispatch attempt. Only the outbox adapter interprets the lease.
type ReservedIntegrationEvent struct {
	Event   IntegrationEvent
	LeaseID string
}

type DispatchResolution uint8

const (
	DispatchPublished DispatchResolution = iota + 1
	// DispatchRetryable means the publisher positively established that no
	// externally visible publish occurred. It is the only retryable result.
	DispatchRetryable
	// DispatchUnknown is terminal until an operator reconciles it. Retrying an
	// ambiguous publish could duplicate an externally visible event.
	DispatchUnknown
)

type PublishOutcome uint8

const (
	PublishConfirmed PublishOutcome = iota + 1
	PublishDefinitelyNotSent
	PublishUnknown
)

// IntegrationOutbox is consumer-owned. Persistence adapters provide atomic
// reservation and lease-checked completion.
type IntegrationOutbox interface {
	ReserveIdentityEvents(context.Context, time.Time, int) ([]ReservedIntegrationEvent, error)
	ResolveIdentityEvent(context.Context, string, string, DispatchResolution, time.Time) error
}

// IntegrationEventPublisher is consumer-owned. Implementations must classify
// ambiguous provider errors as PublishUnknown, never as definitely-not-sent.
type IntegrationEventPublisher interface {
	PublishIdentityEvent(context.Context, IntegrationEvent) (PublishOutcome, error)
}

type IdentityEventDispatcher struct {
	outbox    IntegrationOutbox
	publisher IntegrationEventPublisher
	now       func() time.Time
}

func NewIdentityEventDispatcher(outbox IntegrationOutbox, publisher IntegrationEventPublisher) (*IdentityEventDispatcher, error) {
	if outbox == nil || publisher == nil {
		return nil, errors.New("identity outbox and publisher are required")
	}
	return &IdentityEventDispatcher{outbox: outbox, publisher: publisher, now: time.Now}, nil
}

// Dispatch publishes only records returned after their aggregate transaction
// committed. A provider error without an explicit definitely-not-sent outcome
// is terminal-unknown and is not blindly retried.
func (d *IdentityEventDispatcher) Dispatch(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	now := d.now().UTC()
	reserved, err := d.outbox.ReserveIdentityEvents(ctx, now, identityDispatchBatch)
	if err != nil {
		return err
	}
	var dispatchErr error
	for _, item := range reserved {
		outcome, publishErr := d.publisher.PublishIdentityEvent(ctx, item.Event)
		resolution := DispatchUnknown
		switch outcome {
		case PublishConfirmed:
			resolution = DispatchPublished
		case PublishDefinitelyNotSent:
			resolution = DispatchRetryable
		case PublishUnknown:
			resolution = DispatchUnknown
		default:
			publishErr = errors.Join(publishErr, errors.New("publisher returned invalid outcome"))
		}
		if err := d.outbox.ResolveIdentityEvent(ctx, item.Event.EventID, item.LeaseID, resolution, d.now().UTC()); err != nil {
			dispatchErr = errors.Join(dispatchErr, err)
		}
		if publishErr != nil {
			dispatchErr = errors.Join(dispatchErr, errors.New("identity event publish failed"))
		}
	}
	return dispatchErr
}
