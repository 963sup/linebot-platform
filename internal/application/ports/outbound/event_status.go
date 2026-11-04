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

// EventStatusReader is a read-only operator port. No recovery or LINE send
// methods are exposed. limit=0 omits samples for periodic monitoring.
type EventStatusReader interface {
	Inspect(ctx context.Context, state EventState, limit int, now time.Time) (EventStatusReport, error)
}
