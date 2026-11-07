package messagingdelivery

import (
	"context"
	"errors"
	"strings"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

type ConversationKind string

const (
	ConversationDirect ConversationKind = "direct"
	ConversationGroup  ConversationKind = "group"
	ConversationRoom   ConversationKind = "room"
)

// DeliveryContext is the minimum tenant and conversation scope that must
// survive the asynchronous delivery boundary. Provider identifiers are stored
// only as irreversible SHA-256 fingerprints.
type DeliveryContext struct {
	TenantID, ConnectionID                   string
	ConversationKind                         ConversationKind
	ConversationReferenceSHA256, ActorSHA256 string
}

func (c DeliveryContext) Validate() error {
	if strings.TrimSpace(c.TenantID) == "" || strings.TrimSpace(c.ConnectionID) == "" ||
		!validSHA256(c.ConversationReferenceSHA256) || (c.ActorSHA256 != "" && !validSHA256(c.ActorSHA256)) {
		return errors.New("complete tenant-scoped delivery context is required")
	}
	switch c.ConversationKind {
	case ConversationDirect:
		if c.ActorSHA256 != c.ConversationReferenceSHA256 {
			return errors.New("direct conversation actor must match its address")
		}
	case ConversationGroup, ConversationRoom:
	default:
		return errors.New("supported conversation kind is required")
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

type DeliveryWorkItem struct {
	Key        string
	Context    DeliveryContext
	ReplyToken string
	Message    Message
	ReplyUntil time.Time
	OccurredAt time.Time
	CreatedAt  time.Time
}

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
