package aggregates

import (
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/messaging-delivery/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

var (
	ErrNotClaimable  = errors.New("delivery is not claimable")
	ErrClaimMismatch = errors.New("delivery claim is no longer active")
	ErrTerminal      = errors.New("terminal delivery cannot transition")
)

// Delivery is the consistency boundary for one network send decision.
type Delivery struct {
	id         valueobjects.DeliveryID
	status     valueobjects.Status
	replyUntil time.Time
	claimID    string
	claimUntil time.Time
	version    uint64
	pending    []events.DeliveryEvent
}

func NewDelivery(id valueobjects.DeliveryID, replyUntil time.Time) (Delivery, error) {
	if !id.Valid() || replyUntil.IsZero() {
		return Delivery{}, errors.New("valid delivery ID and reply window are required")
	}
	return Delivery{id: id, status: valueobjects.StatusPending, replyUntil: replyUntil.UTC(), version: 1}, nil
}

func Restore(id valueobjects.DeliveryID, status valueobjects.Status, replyUntil time.Time, claimID string, claimUntil time.Time, version uint64) (Delivery, error) {
	if !id.Valid() || !status.Valid() || replyUntil.IsZero() || version == 0 {
		return Delivery{}, errors.New("invalid delivery snapshot")
	}
	if status == valueobjects.StatusSending && (claimID == "" || claimUntil.IsZero()) {
		return Delivery{}, errors.New("sending delivery requires an active claim")
	}
	if status != valueobjects.StatusSending && (claimID != "" || !claimUntil.IsZero()) {
		return Delivery{}, errors.New("only sending delivery may retain a claim")
	}
	return Delivery{id: id, status: status, replyUntil: replyUntil.UTC(), claimID: claimID, claimUntil: claimUntil.UTC(), version: version}, nil
}

func (d *Delivery) RefreshReplyWindow(until time.Time) error {
	if d.status != valueobjects.StatusPending && d.status != valueobjects.StatusExpired {
		return ErrTerminal
	}
	until = until.UTC()
	if until.IsZero() || !until.After(d.replyUntil) {
		return errors.New("reply window must advance")
	}
	d.replyUntil = until
	d.status = valueobjects.StatusPending
	d.version++
	return nil
}

func (d *Delivery) Claim(claimID string, now, claimUntil time.Time) error {
	if d.status != valueobjects.StatusPending || claimID == "" || claimUntil.IsZero() || !claimUntil.After(now) {
		return ErrNotClaimable
	}
	now = now.UTC()
	if !now.Before(d.replyUntil) {
		d.status = valueobjects.StatusExpired
		d.version++
		return ErrNotClaimable
	}
	d.status, d.claimID, d.claimUntil = valueobjects.StatusSending, claimID, claimUntil.UTC()
	d.version++
	d.pending = append(d.pending, events.Claimed{DeliveryID: d.id, At: now})
	return nil
}

func (d *Delivery) Complete(claimID string, outcome valueobjects.Status, now time.Time) error {
	if d.status.Terminal() {
		return ErrTerminal
	}
	if d.status != valueobjects.StatusSending || claimID == "" || claimID != d.claimID {
		return ErrClaimMismatch
	}
	if outcome != valueobjects.StatusSent && outcome != valueobjects.StatusExpired && outcome != valueobjects.StatusUnknown {
		return errors.New("invalid delivery outcome")
	}
	d.status, d.claimID, d.claimUntil = outcome, "", time.Time{}
	d.version++
	now = now.UTC()
	d.pending = append(d.pending, events.Completed{DeliveryID: d.id, Outcome: outcome, At: now})
	if outcome == valueobjects.StatusUnknown {
		d.pending = append(d.pending, events.OutcomeUnknown{DeliveryID: d.id, At: now})
	}
	return nil
}

// Reconcile expires unsent work and converts abandoned sends to unknown.
func (d *Delivery) Reconcile(now time.Time) bool {
	now = now.UTC()
	switch {
	case d.status == valueobjects.StatusPending && !now.Before(d.replyUntil):
		d.status = valueobjects.StatusExpired
	case d.status == valueobjects.StatusSending && !now.Before(d.claimUntil):
		d.status, d.claimID, d.claimUntil = valueobjects.StatusUnknown, "", time.Time{}
		d.pending = append(d.pending, events.OutcomeUnknown{DeliveryID: d.id, At: now})
	default:
		return false
	}
	d.version++
	return true
}

func (d Delivery) ID() valueobjects.DeliveryID { return d.id }
func (d Delivery) Status() valueobjects.Status { return d.status }
func (d Delivery) ReplyUntil() time.Time       { return d.replyUntil }
func (d Delivery) ClaimID() string             { return d.claimID }
func (d Delivery) ClaimUntil() time.Time       { return d.claimUntil }
func (d Delivery) Version() uint64             { return d.version }
func (d Delivery) PendingEvents() []events.DeliveryEvent {
	return append([]events.DeliveryEvent(nil), d.pending...)
}

// AcknowledgeEvents clears events after the aggregate state and event records
// have been committed successfully by the application boundary.
func (d *Delivery) AcknowledgeEvents() {
	if d != nil {
		d.pending = nil
	}
}
