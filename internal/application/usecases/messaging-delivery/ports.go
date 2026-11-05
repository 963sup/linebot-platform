package messagingdelivery

import (
	"context"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

type DeliveryWorkItem struct {
	Key        string
	ReplyToken string
	Message    Message
	ReplyUntil time.Time
	OccurredAt time.Time
	CreatedAt  time.Time
}

// EventJob is retained for source compatibility.
// Deprecated: use DeliveryWorkItem.
type EventJob = DeliveryWorkItem

type DeliveryState = valueobjects.Status

const (
	DeliverySent    = valueobjects.StatusSent
	DeliveryExpired = valueobjects.StatusExpired
	DeliveryUnknown = valueobjects.StatusUnknown
)

// EventRecorder is the persistence capability needed while accepting a
// verified delivery intent.
type EventRecorder interface {
	Record(context.Context, DeliveryWorkItem) (bool, error)
}

// DeliveryClaimer is the persistence capability needed by a delivery worker.
type DeliveryClaimer interface {
	Claim(context.Context, string, string, time.Time) (DeliveryWorkItem, bool, error)
	Complete(context.Context, string, string, DeliveryState, time.Time) error
}

// DeliveryRecovery is the persistence capability needed by reconciliation.
type DeliveryRecovery interface {
	Recover(context.Context, time.Time, int) ([]string, error)
}

// DeliveryWorkflowStore is the complete transactional persistence boundary for
// durable delivery. Consumers should depend on the narrower capability above.
type DeliveryWorkflowStore interface {
	EventRecorder
	DeliveryClaimer
	DeliveryRecovery
}

// EventStore is retained for source compatibility.
// Deprecated: use DeliveryWorkflowStore or a narrower consumer-owned port.
type EventStore = DeliveryWorkflowStore

// EventState is the read-model name for the canonical Domain delivery status.
// It remains an alias for source compatibility; it is not a second state machine.
type EventState = valueobjects.Status

const (
	EventPending = valueobjects.StatusPending
	EventSending = valueobjects.StatusSending
	EventSent    = valueobjects.StatusSent
	EventExpired = valueobjects.StatusExpired
	EventUnknown = valueobjects.StatusUnknown
)

type EventStatusSample struct {
	KeyHint   string
	State     EventState
	UpdatedAt time.Time
}

type EventStatusReport struct {
	GeneratedAt             time.Time
	Counts                  map[EventState]int64
	OldestPendingAgeSeconds int64
	Samples                 []EventStatusSample
}

type EventStatusReader interface {
	Inspect(context.Context, EventState, int, time.Time) (EventStatusReport, error)
}
