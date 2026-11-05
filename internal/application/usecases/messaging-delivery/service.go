// Package messagingdelivery orchestrates durable event receipt and delivery
// without depending on Firestore, Cloud Tasks, HTTP or any provider SDK.
package messagingdelivery

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

const (
	replyWindow  = 45 * time.Second
	maxEventAge  = 20 * time.Minute
	maxClockSkew = 5 * time.Minute
	maxDispatch  = 100
)

// ErrEventTooOld indicates no LINE Reply can be attempted because the
// original webhook event is outside the documented redelivery eligibility.
var ErrEventTooOld = errors.New("LINE event reply window expired")

// ErrDeliveryInFlight asks the task transport to retry later. It means a
// previous worker committed the one-use send claim but has not yet durably
// recorded a terminal outcome; acknowledging that retry would discard the
// only remaining reconciliation trigger.
var ErrDeliveryInFlight = errors.New("LINE delivery is still in flight")

// EventKey is retained as the Application name for the Domain delivery ID.
type EventKey = valueobjects.DeliveryID

// NewEventKey creates a stable key for a LINE channel + webhook event ID across
// redeliveries. The opaque hash avoids storing either identifier in document paths.
func NewEventKey(channel, eventID string) (EventKey, error) {
	if strings.TrimSpace(channel) == "" || strings.TrimSpace(eventID) == "" {
		return EventKey{}, errors.New("event channel and ID are required")
	}
	hash := sha256.Sum256([]byte(channel + "\x00" + eventID))
	return valueobjects.ParseDeliveryID(hex.EncodeToString(hash[:]))
}

// ParseEventKey validates an event key received from a persistence or transport
// boundary before it enters the application workflow.
func ParseEventKey(value string) (EventKey, error) {
	return valueobjects.ParseDeliveryID(value)
}

type EventQueue interface {
	Enqueue(context.Context, EventKey) error
}

type Ingestor struct {
	store EventRecorder
	queue EventQueue
	now   func() time.Time
}

func NewIngestor(store EventRecorder, queue EventQueue) (*Ingestor, error) {
	if store == nil || queue == nil {
		return nil, errors.New("event store and queue are required")
	}
	return &Ingestor{store: store, queue: queue, now: time.Now}, nil
}

// Ingest must only receive events after LINE webhook signature verification.
// Queue is called even for duplicates: this repairs a crash after the Firestore
// commit but before CreateTask. A separate dispatcher repairs missed enqueueing
// when no LINE redelivery arrives.
func (s *Ingestor) Ingest(ctx context.Context, channel, eventID, replyToken string, occurredAt time.Time, message Message) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(replyToken) == "" {
		return errors.New("reply token is required")
	}
	if err := message.Validate(); err != nil {
		return err
	}
	key, err := NewEventKey(channel, eventID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	occurredAt = occurredAt.UTC()
	if occurredAt.IsZero() || occurredAt.After(now.Add(maxClockSkew)) {
		return errors.New("invalid LINE event timestamp")
	}
	// A redelivered reply token may be usable for up to 20 minutes from
	// the original event, and only shortly after each webhook receipt.
	// This is a conservative local upper bound, not a delivery guarantee.
	if !now.Before(occurredAt.Add(maxEventAge)) {
		return ErrEventTooOld
	}
	replyUntil := now.Add(replyWindow)
	if eventLimit := occurredAt.Add(maxEventAge); replyUntil.After(eventLimit) {
		replyUntil = eventLimit
	}
	if _, err := s.store.Record(ctx, DeliveryWorkItem{
		Key: key.String(), ReplyToken: replyToken, Message: message,
		CreatedAt: now, OccurredAt: occurredAt, ReplyUntil: replyUntil,
	}); err != nil {
		return err
	}
	return s.queue.Enqueue(ctx, key)
}

type Processor struct {
	store  DeliveryClaimer
	sender interface {
		Reply(context.Context, string, []Message) error
	}
	now func() time.Time
}

func NewProcessor(store DeliveryClaimer, sender interface {
	Reply(context.Context, string, []Message) error
}) (*Processor, error) {
	if store == nil || sender == nil {
		return nil, errors.New("event store and messaging port are required")
	}
	return &Processor{store: store, sender: sender, now: time.Now}, nil
}

// Process runs at most one LINE reply attempt for a given event key. A second
// Cloud Tasks delivery will never call LINE again once a lease was acquired.
// The sending/unknown states require explicit reconciliation, not auto-retry.
func (p *Processor) Process(ctx context.Context, key EventKey) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !key.Valid() {
		return errors.New("invalid event key")
	}
	var lease [16]byte
	if _, err := rand.Read(lease[:]); err != nil {
		return err
	}
	leaseID := hex.EncodeToString(lease[:])
	job, claimed, err := p.store.Claim(ctx, key.String(), leaseID, p.now().UTC())
	if err != nil || !claimed {
		return err
	}
	// Do not automatically retry any ambiguous LINE result. An unknown reply
	// must be reviewed; sending it again could deliver a duplicate.
	outcome := DeliverySent
	if !p.now().Before(job.ReplyUntil) {
		outcome = DeliveryExpired
	} else if err := p.sender.Reply(ctx, job.ReplyToken, []Message{job.Message}); err != nil {
		outcome = DeliveryUnknown
	}
	return p.store.Complete(ctx, key.String(), leaseID, outcome)
}

type Dispatcher struct {
	store DeliveryRecovery
	queue EventQueue
	now   func() time.Time
}

func NewDispatcher(store DeliveryRecovery, queue EventQueue) (*Dispatcher, error) {
	if store == nil || queue == nil {
		return nil, errors.New("event store and queue are required")
	}
	return &Dispatcher{store: store, queue: queue, now: time.Now}, nil
}

// Dispatch re-enqueues pending work and marks abandoned sending leases unknown
// via the storage adapter. Run it on an authenticated schedule. No pending work
// is deleted on CreateTask errors; the next run can retry safely.
func (d *Dispatcher) Dispatch(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	keys, err := d.store.Recover(ctx, d.now().UTC(), maxDispatch)
	if err != nil {
		return err
	}
	// Attempt every reserved item. A single queue error must not prevent
	// younger work from being offered; Recover will retry after cooldown.
	var dispatchErr error
	for _, value := range keys {
		key, err := ParseEventKey(value)
		if err != nil {
			dispatchErr = errors.Join(dispatchErr, err)
			continue
		}
		if err := d.queue.Enqueue(ctx, key); err != nil {
			dispatchErr = errors.Join(dispatchErr, err)
		}
	}
	return dispatchErr
}
