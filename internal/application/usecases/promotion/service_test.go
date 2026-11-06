package promotion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/promotion/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
)

var errPromotionPort = errors.New("promotion port failure")

type promotionRepository struct {
	value     aggregates.Promotion
	findErr   error
	saveErr   error
	saveCalls int
}

func (r *promotionRepository) Find(context.Context, valueobjects.PromotionID) (aggregates.Promotion, error) {
	return r.value, r.findErr
}
func (r *promotionRepository) Save(_ context.Context, value aggregates.Promotion) error {
	r.saveCalls++
	if r.saveErr == nil {
		r.value = value
	}
	return r.saveErr
}

func TestPromotionCommandsPersistLifecycle(t *testing.T) {
	t.Parallel()
	repository := &promotionRepository{}
	service, _ := NewService(repository)
	id, offer, policy, period, now := promotionValues(t)
	defined, err := service.Define(context.Background(), id, offer, policy, period)
	if err != nil || defined.Status() != aggregates.StatusDraft {
		t.Fatalf("Define() = (%v, %v), want draft, nil", defined.Status(), err)
	}
	active, err := service.Activate(context.Background(), id, now)
	if err != nil || active.Status() != aggregates.StatusActive {
		t.Fatalf("Activate() = (%v, %v), want active, nil", active.Status(), err)
	}
	closed, err := service.Close(context.Background(), id)
	if err != nil || closed.Status() != aggregates.StatusClosed {
		t.Fatalf("Close() = (%v, %v), want closed, nil", closed.Status(), err)
	}
	if repository.saveCalls != 3 {
		t.Fatalf("Save calls = %d, want 3", repository.saveCalls)
	}
}

func TestCreateDeliveryIntentPersistsRevisionWithoutRemoteSideEffect(t *testing.T) {
	t.Parallel()
	id, _, policy, _, now := promotionValues(t)
	promotion := activePromotion(t)
	repository := &promotionRepository{value: promotion}
	service, _ := NewService(repository)
	decision, _ := valueobjects.NewEligibilityDecision(policy, true)
	intent, err := service.CreateDeliveryIntent(context.Background(), id, "recipient-1", decision, now)
	if err != nil || !intent.Valid() {
		t.Fatalf("CreateDeliveryIntent() = (%v, %v), want valid intent", intent, err)
	}
	if repository.value.Revision() != promotion.Revision()+1 {
		t.Fatal("delivery intent did not persist the advanced promotion revision")
	}
	if repository.saveCalls != 1 {
		t.Fatalf("Save calls = %d, want 1", repository.saveCalls)
	}

	repository.saveErr = errPromotionPort
	if _, err := service.CreateDeliveryIntent(context.Background(), id, "recipient-2", decision, now); !errors.Is(err, errPromotionPort) {
		t.Fatalf("CreateDeliveryIntent(save error) = %v, want %v", err, errPromotionPort)
	}
}

func TestCreateDeliveryIntentPropagatesDomainErrors(t *testing.T) {
	t.Parallel()
	id, _, policy, _, now := promotionValues(t)
	repository := &promotionRepository{value: activePromotion(t)}
	service, _ := NewService(repository)
	ineligible, _ := valueobjects.NewEligibilityDecision(policy, false)
	if _, err := service.CreateDeliveryIntent(context.Background(), id, "recipient", ineligible, now); !errors.Is(err, aggregates.ErrIneligibleRecipient) {
		t.Fatalf("CreateDeliveryIntent(ineligible) = %v, want %v", err, aggregates.ErrIneligibleRecipient)
	}
	if repository.saveCalls != 0 {
		t.Fatal("invalid domain decision caused side effects")
	}
}

func TestPromotionServiceDependenciesContextAndRepositoryErrors(t *testing.T) {
	t.Parallel()
	repository := &promotionRepository{}
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil")
	}
	service, _ := NewService(repository)
	id, offer, policy, period, now := promotionValues(t)
	if _, err := service.Define(nil, id, offer, policy, period); !errors.Is(err, ErrContextRequired) {
		t.Fatalf("Define(nil) = %v, want %v", err, ErrContextRequired)
	}
	repository.findErr = errPromotionPort
	if _, err := service.Activate(context.Background(), id, now); !errors.Is(err, errPromotionPort) {
		t.Fatalf("Activate(find error) = %v, want %v", err, errPromotionPort)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Close(ctx, id); !errors.Is(err, context.Canceled) {
		t.Fatalf("Close(canceled) = %v, want %v", err, context.Canceled)
	}
}

func promotionValues(t *testing.T) (valueobjects.PromotionID, valueobjects.OfferReference, valueobjects.EligibilityPolicyReference, valueobjects.Period, time.Time) {
	t.Helper()
	id, _ := valueobjects.NewPromotionID("spring")
	offer, _ := valueobjects.NewOfferReference("offer-1")
	policy, _ := valueobjects.NewEligibilityPolicyReference("members")
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	period, err := valueobjects.NewPeriod(now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return id, offer, policy, period, now
}

func activePromotion(t *testing.T) aggregates.Promotion {
	t.Helper()
	id, offer, policy, period, now := promotionValues(t)
	promotion, err := aggregates.NewPromotion(id, offer, policy, period)
	if err != nil {
		t.Fatal(err)
	}
	if err := promotion.Activate(now); err != nil {
		t.Fatal(err)
	}
	return promotion
}
