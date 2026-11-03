// Package ingestion orchestrates durable LINE event receipt and delivery
// without depending on Firestore, Cloud Tasks, HTTP or any provider SDK.
package ingestion

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

const (
	replyWindow = 45 * time.Second
	maxDispatch = 100
)

// EventKey is stable for a LINE channel + webhook event ID across redeliveries.
// The opaque hash avoids storing either identifier in Firestore document paths.
func EventKey(channel, eventID string) (string, error) {
	if strings.TrimSpace(channel) == "" || strings.TrimSpace(eventID) == "" {
		return "", errors.New("event channel and ID are required")
	}
	hash := sha256.Sum256([]byte(channel + "\x00" + eventID))
	return hex.EncodeToString(hash[:]), nil
}

type Ingestor struct {
	store outbound.EventStore
	queue outbound.EventQueue
	now   func() time.Time
}

func NewIngestor(store outbound.EventStore, queue outbound.EventQueue) (*Ingestor, error) {
	if store == nil || queue == nil {
		return nil, errors.New("event store and queue are required")
	}
	return &Ingestor{store: store, queue: queue, now: time.Now}, nil
}

// Ingest must only receive events after LINE webhook signature verification.
// Queue is called even for duplicates: this repairs a crash after the Firestore
// commit but before CreateTask. A separate dispatcher repairs missed enqueueing
// when no LINE redelivery arrives.
func (s *Ingestor) Ingest(ctx context.Context, channel, eventID, replyToken string, message outbound.Message) error {
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
	key, err := EventKey(channel, eventID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	if _, err := s.store.Record(ctx, outbound.EventJob{
		Key: key, ReplyToken: replyToken, Message: message,
		CreatedAt: now, ReplyUntil: now.Add(replyWindow),
	}); err != nil {
		return err
	}
	return s.queue.Enqueue(ctx, key)
}

type Processor struct {
	store  outbound.EventStore
	sender outbound.Messaging
	now    func() time.Time
}

func NewProcessor(store outbound.EventStore, sender outbound.Messaging) (*Processor, error) {
	if store == nil || sender == nil {
		return nil, errors.New("event store and messaging port are required")
	}
	return &Processor{store: store, sender: sender, now: time.Now}, nil
}

// Process runs at most one LINE reply attempt for a given event key. A second
// Cloud Tasks delivery will never call LINE again once a lease was acquired.
// The sending/unknown states require explicit reconciliation, not auto-retry.
func (p *Processor) Process(ctx context.Context, key string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !ValidKey(key) {
		return errors.New("invalid event key")
	}
	var lease [16]byte
	if _, err := rand.Read(lease[:]); err != nil {
		return err
	}
	leaseID := hex.EncodeToString(lease[:])
	job, claimed, err := p.store.Claim(ctx, key, leaseID, p.now().UTC())
	if err != nil || !claimed {
		return err
	}
	// Do not automatically retry any ambiguous LINE result. An unknown reply
	// must be reviewed; sending it again could deliver a duplicate.
	outcome := outbound.DeliverySent
	if p.now().After(job.ReplyUntil) {
		outcome = outbound.DeliveryExpired
	} else if err := p.sender.Reply(ctx, job.ReplyToken, []outbound.Message{job.Message}); err != nil {
		outcome = outbound.DeliveryUnknown
	}
	return p.store.Complete(ctx, key, leaseID, outcome)
}

// ValidKey constrains all task/body identifiers to opaque SHA-256 hex strings.
func ValidKey(key string) bool {
	if len(key) != 64 {
		return false
	}
	for _, c := range key {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

type Dispatcher struct {
	store outbound.EventStore
	queue outbound.EventQueue
	now   func() time.Time
}

func NewDispatcher(store outbound.EventStore, queue outbound.EventQueue) (*Dispatcher, error) {
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
	for _, key := range keys {
		if err := d.queue.Enqueue(ctx, key); err != nil {
			return err
		}
	}
	return nil
}
