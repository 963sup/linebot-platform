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
	Complete(context.Context, string, string, DeliveryState) error
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

type EventState string

const (
	EventPending EventState = "pending"
	EventSending EventState = "sending"
	EventSent    EventState = "sent"
	EventExpired EventState = "expired"
	EventUnknown EventState = "unknown"
)

func (s EventState) Valid() bool {
	switch s {
	case EventPending, EventSending, EventSent, EventExpired, EventUnknown:
		return true
	default:
		return false
	}
}

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
