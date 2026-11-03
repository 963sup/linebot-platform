package outbound

import (
	"context"
	"time"
)

// EventJob contains only the derived response and the short-lived LINE reply
// token. It never stores the original LINE message, user ID or raw webhook.
type EventJob struct {
	Key        string
	ReplyToken string
	Message    Message
	ReplyUntil time.Time
	OccurredAt time.Time
	CreatedAt  time.Time
}

type DeliveryState string

const (
	DeliverySent    DeliveryState = "sent"
	DeliveryExpired DeliveryState = "expired"
	DeliveryUnknown DeliveryState = "unknown"
)

// EventStore provides a durable, transaction-backed inbox/outbox boundary.
// Record MUST atomically create both documents or create neither, and repeated
// keys MUST NOT overwrite the original response payload. A fresh signed
// redelivery may extend a never-sent pending/expired reply window, bounded by
// the original event age; sent/sending/unknown states must never be reopened.
// Claim is a single-use atomic transition from pending to sending.
// Sending is never automatically reclaimed
// for a second LINE API call because a remote timeout may mean it was sent.
type EventStore interface {
	Record(ctx context.Context, job EventJob) (created bool, err error)
	Claim(ctx context.Context, key, leaseID string, now time.Time) (EventJob, bool, error)
	Complete(ctx context.Context, key, leaseID string, outcome DeliveryState) error
	Recover(ctx context.Context, now time.Time, limit int) ([]string, error)
}

// EventQueue enqueues only a deterministic, non-sensitive outbox key. It MUST
// tolerate a duplicate task name and MUST NOT interpret an ambiguous RPC
// failure as proof no task was created.
type EventQueue interface {
	Enqueue(ctx context.Context, key string) error
}
