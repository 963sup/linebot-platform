package aggregates_test

import (
	"errors"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/promotion/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/promotion/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
)

var (
	promotionStart = time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	promotionEnd   = promotionStart.Add(24 * time.Hour)
)

func TestPromotionLifecycle(t *testing.T) {
	t.Parallel()

	promotion := newPromotion(t)
	if got, want := promotion.Status(), aggregates.StatusDraft; got != want {
		t.Errorf("NewPromotion(...).Status() = %v, want %v", got, want)
	}
	if err := promotion.Activate(promotionStart); err != nil {
		t.Fatalf("Promotion.Activate(%v) error = %v, want nil", promotionStart, err)
	}
	if got, want := promotion.Status(), aggregates.StatusActive; got != want {
		t.Errorf("Promotion.Status() after Activate = %v, want %v", got, want)
	}
	if err := promotion.Close(); err != nil {
		t.Fatalf("Promotion.Close() error = %v, want nil", err)
	}
	if got, want := promotion.Status(), aggregates.StatusClosed; got != want {
		t.Errorf("Promotion.Status() after Close = %v, want %v", got, want)
	}
}

func TestClosedPromotionCannotCreateDeliveryIntent(t *testing.T) {
	t.Parallel()

	promotion := activePromotion(t)
	if err := promotion.Close(); err != nil {
		t.Fatalf("Promotion.Close() error = %v, want nil", err)
	}
	decision := eligibleDecision(t, "members-v1")
	if _, err := promotion.CreateDeliveryIntent("line:user:1", decision, promotionStart); !errors.Is(err, aggregates.ErrPromotionClosed) {
		t.Errorf("Promotion.CreateDeliveryIntent(...) after Close error = %v, want %v", err, aggregates.ErrPromotionClosed)
	}
}

func TestActivePromotionCreatesDeliveryIntent(t *testing.T) {
	t.Parallel()

	promotion := activePromotion(t)
	intent, err := promotion.CreateDeliveryIntent("line:user:1", eligibleDecision(t, "members-v1"), promotionStart.Add(time.Hour))
	if err != nil {
		t.Fatalf("Promotion.CreateDeliveryIntent(...) error = %v, want nil", err)
	}
	if got, want := intent.RecipientReference(), "line:user:1"; got != want {
		t.Errorf("DeliveryIntent.RecipientReference() = %q, want %q", got, want)
	}
	gotEvents := promotion.PendingEvents()
	if _, ok := gotEvents[len(gotEvents)-1].(events.DeliveryIntentCreated); !ok {
		t.Errorf("Promotion.PendingEvents() last event = %T, want events.DeliveryIntentCreated", gotEvents[len(gotEvents)-1])
	}
}

func TestPromotionRejectsUnapprovedDeliveryIntents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decision valueobjects.EligibilityDecision
		at       time.Time
		wantErr  error
	}{
		{name: "ineligible", decision: ineligibleDecision(t, "members-v1"), at: promotionStart, wantErr: aggregates.ErrIneligibleRecipient},
		{name: "wrong policy", decision: eligibleDecision(t, "other-policy"), at: promotionStart, wantErr: aggregates.ErrEligibilityPolicyMismatch},
		{name: "outside period", decision: eligibleDecision(t, "members-v1"), at: promotionEnd, wantErr: aggregates.ErrOutsidePromotionPeriod},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			promotion := activePromotion(t)
			if _, err := promotion.CreateDeliveryIntent("line:user:1", tt.decision, tt.at); !errors.Is(err, tt.wantErr) {
				t.Errorf("Promotion.CreateDeliveryIntent(..., %v) error = %v, want %v", tt.at, err, tt.wantErr)
			}
		})
	}
}

func TestPromotionRejectsInvalidTransitionsAndActivationTime(t *testing.T) {
	t.Parallel()

	promotion := newPromotion(t)
	if err := promotion.Close(); !errors.Is(err, aggregates.ErrInvalidPromotionTransition) {
		t.Errorf("Promotion.Close() from draft error = %v, want %v", err, aggregates.ErrInvalidPromotionTransition)
	}
	if err := promotion.Activate(promotionStart.Add(-time.Second)); !errors.Is(err, aggregates.ErrOutsidePromotionPeriod) {
		t.Errorf("Promotion.Activate(before period) error = %v, want %v", err, aggregates.ErrOutsidePromotionPeriod)
	}
}

func TestPromotionEventsAreDetached(t *testing.T) {
	t.Parallel()

	promotion := newPromotion(t)
	got := promotion.PendingEvents()
	got[0] = nil
	if promotion.PendingEvents()[0] == nil {
		t.Error("Promotion.PendingEvents() returned mutable internal storage, want detached slice")
	}
	promotion.AcknowledgeEvents()
	if len(promotion.PendingEvents()) != 0 {
		t.Error("Promotion.AcknowledgeEvents() did not clear committed events")
	}
}

func TestPromotionRevisionAdvancesOnlyOnSuccessfulMutations(t *testing.T) {
	t.Parallel()

	promotion := newPromotion(t)
	if got := promotion.Revision(); got != 1 {
		t.Fatalf("Promotion.Revision() = %d, want 1", got)
	}
	if err := promotion.Close(); !errors.Is(err, aggregates.ErrInvalidPromotionTransition) {
		t.Fatalf("Promotion.Close() error = %v, want invalid transition", err)
	}
	if got := promotion.Revision(); got != 1 {
		t.Fatalf("rejected transition revision = %d, want 1", got)
	}
	if err := promotion.Activate(promotionStart); err != nil {
		t.Fatal(err)
	}
	if got := promotion.Revision(); got != 2 {
		t.Fatalf("activated revision = %d, want 2", got)
	}
	if err := promotion.Close(); err != nil {
		t.Fatal(err)
	}
	if got := promotion.Revision(); got != 3 {
		t.Fatalf("closed revision = %d, want 3", got)
	}
}

func TestRestorePromotionPreservesRevisionWithoutEvents(t *testing.T) {
	t.Parallel()

	promotion := newPromotion(t)
	restored, err := aggregates.RestorePromotion(
		promotion.ID(),
		promotion.Offer(),
		promotion.EligibilityPolicy(),
		promotion.Period(),
		promotion.Status(),
		7,
	)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Revision() != 7 || len(restored.PendingEvents()) != 0 {
		t.Fatalf("restored promotion revision/events = %d/%d, want 7/0", restored.Revision(), len(restored.PendingEvents()))
	}
}

func newPromotion(t *testing.T) aggregates.Promotion {
	t.Helper()
	id, err := valueobjects.NewPromotionID("new-year")
	if err != nil {
		t.Fatalf("NewPromotionID(...) error = %v, want nil", err)
	}
	offer, err := valueobjects.NewOfferReference("offer:new-year")
	if err != nil {
		t.Fatalf("NewOfferReference(...) error = %v, want nil", err)
	}
	policy, err := valueobjects.NewEligibilityPolicyReference("members-v1")
	if err != nil {
		t.Fatalf("NewEligibilityPolicyReference(...) error = %v, want nil", err)
	}
	period, err := valueobjects.NewPeriod(promotionStart, promotionEnd)
	if err != nil {
		t.Fatalf("NewPeriod(...) error = %v, want nil", err)
	}
	promotion, err := aggregates.NewPromotion(id, offer, policy, period)
	if err != nil {
		t.Fatalf("NewPromotion(...) error = %v, want nil", err)
	}
	return promotion
}

func activePromotion(t *testing.T) aggregates.Promotion {
	t.Helper()
	promotion := newPromotion(t)
	if err := promotion.Activate(promotionStart); err != nil {
		t.Fatalf("Promotion.Activate(%v) error = %v, want nil", promotionStart, err)
	}
	return promotion
}

func eligibleDecision(t *testing.T, policyName string) valueobjects.EligibilityDecision {
	t.Helper()
	return decision(t, policyName, true)
}

func ineligibleDecision(t *testing.T, policyName string) valueobjects.EligibilityDecision {
	t.Helper()
	return decision(t, policyName, false)
}

func decision(t *testing.T, policyName string, eligible bool) valueobjects.EligibilityDecision {
	t.Helper()
	policy, err := valueobjects.NewEligibilityPolicyReference(policyName)
	if err != nil {
		t.Fatalf("NewEligibilityPolicyReference(%q) error = %v, want nil", policyName, err)
	}
	got, err := valueobjects.NewEligibilityDecision(policy, eligible)
	if err != nil {
		t.Fatalf("NewEligibilityDecision(...) error = %v, want nil", err)
	}
	return got
}
