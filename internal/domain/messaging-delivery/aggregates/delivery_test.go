package aggregates

import (
	"errors"
	"testing"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

func delivery(t *testing.T) Delivery {
	t.Helper()
	id, err := valueobjects.ParseDeliveryID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDelivery(id, time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDeliveryLifecycle(t *testing.T) {
	d := delivery(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := d.Claim("lease", now, now.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := d.Claim("other", now, now.Add(time.Minute)); !errors.Is(err, ErrNotClaimable) {
		t.Fatalf("second claim: %v", err)
	}
	if err := d.Complete("other", valueobjects.StatusSent, now); !errors.Is(err, ErrClaimMismatch) {
		t.Fatalf("wrong claim: %v", err)
	}
	if err := d.Complete("lease", valueobjects.StatusSent, now); err != nil {
		t.Fatal(err)
	}
	if err := d.Complete("lease", valueobjects.StatusSent, now); !errors.Is(err, ErrTerminal) {
		t.Fatalf("terminal reopened: %v", err)
	}
	if d.Status() != valueobjects.StatusSent || d.Version() != 3 || d.ClaimID() != "" || !d.ClaimUntil().IsZero() {
		t.Fatal("completed delivery retained a claim or lost its revision")
	}
}

func TestRestoreDeliveryPreservesState(t *testing.T) {
	d := delivery(t)
	restored, err := Restore(d.ID(), d.Status(), d.ReplyUntil(), d.ClaimID(), d.ClaimUntil(), d.Version())
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID() != d.ID() || restored.Status() != d.Status() || restored.ReplyUntil() != d.ReplyUntil() || restored.Version() != d.Version() {
		t.Fatal("Restore changed delivery state")
	}
}

func TestDeliveryReconcileAndRefresh(t *testing.T) {
	d := delivery(t)
	late := d.ReplyUntil().Add(time.Second)
	if !d.Reconcile(late) || d.Status() != valueobjects.StatusExpired {
		t.Fatal("pending delivery did not expire")
	}
	if err := d.RefreshReplyWindow(late.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if d.Status() != valueobjects.StatusPending {
		t.Fatal("expired delivery did not reopen before a send")
	}
	if err := d.Claim("lease", late, late.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if !d.Reconcile(late.Add(2*time.Second)) || d.Status() != valueobjects.StatusUnknown {
		t.Fatal("abandoned send was not unknown")
	}
	if err := d.RefreshReplyWindow(late.Add(2 * time.Minute)); !errors.Is(err, ErrTerminal) {
		t.Fatalf("unknown delivery reopened: %v", err)
	}
}

func TestDeliveryRejectsZeroAuthoritativeTimeWithoutMutation(t *testing.T) {
	d := delivery(t)
	initialVersion := d.Version()
	if err := d.Claim("lease", time.Time{}, d.ReplyUntil().Add(time.Second)); !errors.Is(err, ErrNotClaimable) {
		t.Fatalf("zero-time claim error=%v, want ErrNotClaimable", err)
	}
	if d.Version() != initialVersion || d.Status() != valueobjects.StatusPending {
		t.Fatal("zero-time claim mutated delivery")
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := d.Claim("lease", now, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	claimedVersion := d.Version()
	claimedUntil := d.ClaimUntil()
	if err := d.Complete("lease", valueobjects.StatusSent, time.Time{}); err == nil {
		t.Fatal("zero-time completion succeeded")
	}
	if d.Version() != claimedVersion || d.Status() != valueobjects.StatusSending || d.ClaimID() != "lease" || d.ClaimUntil() != claimedUntil {
		t.Fatal("zero-time completion mutated delivery")
	}
	if d.Reconcile(time.Time{}) {
		t.Fatal("zero-time reconcile reported a transition")
	}
	if d.Version() != claimedVersion || d.Status() != valueobjects.StatusSending || d.ClaimID() != "lease" || d.ClaimUntil() != claimedUntil {
		t.Fatal("zero-time reconcile mutated delivery")
	}
}
