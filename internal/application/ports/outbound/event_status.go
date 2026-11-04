package outbound

import (
	"context"
	"time"
)

// EventState describes infrastructure delivery state, not a domain aggregate.
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

// EventStatusSample intentionally contains no LINE token, user ID or message.
type EventStatusSample struct {
	KeyHint   string     `json:"key_hint"`
	State     EventState `json:"state"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type EventStatusReport struct {
	GeneratedAt             time.Time            `json:"generated_at"`
	Counts                  map[EventState]int64 `json:"counts"`
	OldestPendingAgeSeconds int64                `json:"oldest_pending_age_seconds"`
	Samples                 []EventStatusSample  `json:"samples,omitempty"`
}

// EventStatusReader is a read-only operator port. No recovery or LINE send
// methods are exposed. limit=0 omits samples for periodic monitoring.
type EventStatusReader interface {
	Inspect(ctx context.Context, state EventState, limit int, now time.Time) (EventStatusReport, error)
}
