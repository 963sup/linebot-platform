package firestore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/promotion/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewPromotionRepositoryRequiresClient(t *testing.T) {
	if _, err := NewPromotionRepository(nil); err == nil {
		t.Fatal("NewPromotionRepository(nil) error = nil")
	}
}

func TestPromotionRepositoryMappingRoundTripDoesNotRaiseEvents(t *testing.T) {
	value := newTestPromotion(t)
	now := time.Unix(1_700_000_000, 0).UTC()
	stored := storePromotion(value, now)
	stored.CreatedAt = now

	restored, err := restorePromotion(stored)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID() != value.ID() || restored.Offer() != value.Offer() || restored.EligibilityPolicy() != value.EligibilityPolicy() ||
		restored.Status() != value.Status() || restored.Revision() != value.Revision() ||
		!restored.Period().Start().Equal(value.Period().Start()) || !restored.Period().End().Equal(value.Period().End()) {
		t.Fatalf("restored promotion = %#v, want %#v", restored, value)
	}
}

func TestPromotionRepositoryRevisionGate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	value := newTestPromotion(t)
	current := storePromotion(value, now)
	current.CreatedAt = now

	if err := value.Activate(value.Period().Start()); err != nil {
		t.Fatal(err)
	}
	next := storePromotion(value, now.Add(time.Minute))
	next.CreatedAt = current.CreatedAt
	if err := validatePromotionRevision(current, next); err != nil {
		t.Fatalf("next revision error = %v", err)
	}

	if err := validatePromotionRevision(next, next); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("stale revision error = %v, want ErrConcurrentUpdate", err)
	}
	if err := validatePromotionRevision(current, current); !errors.Is(err, ErrRepositoryConflict) {
		t.Fatalf("duplicate create error = %v, want ErrRepositoryConflict", err)
	}

	corrupt := current
	corrupt.SchemaVersion++
	if err := validatePromotionRevision(corrupt, next); !errors.Is(err, ErrRepositoryCorrupt) {
		t.Fatalf("corrupt state error = %v, want ErrRepositoryCorrupt", err)
	}
}

func TestPromotionRepositoryRejectsInvalidDocuments(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	stored := storePromotion(newTestPromotion(t), now)
	stored.CreatedAt = now

	tests := map[string]func(*storedPromotion){
		"schema":         func(value *storedPromotion) { value.SchemaVersion = 2 },
		"revision":       func(value *storedPromotion) { value.Revision = 0 },
		"created at":     func(value *storedPromotion) { value.CreatedAt = time.Time{} },
		"updated before": func(value *storedPromotion) { value.UpdatedAt = value.CreatedAt.Add(-time.Second) },
		"status":         func(value *storedPromotion) { value.Status = 255 },
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			value := stored
			corrupt(&value)
			if _, err := restorePromotion(value); err == nil {
				t.Fatal("restorePromotion(corrupt) error = nil")
			}
		})
	}
}

func TestPromotionRepositoryErrorClassification(t *testing.T) {
	if err := classifyPromotionSaveError(context.Background(), status.Error(codes.Aborted, "provider detail")); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("aborted error = %v, want ErrConcurrentUpdate", err)
	}
	if err := classifyPromotionSaveError(context.Background(), status.Error(codes.PermissionDenied, "secret project")); !errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("permission error = %v, want ErrRepositoryUnavailable", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := classifyPromotionSaveError(ctx, status.Error(codes.Canceled, "provider detail")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v, want context.Canceled", err)
	}
}

func TestPromotionDocumentIDIsOpaqueAndStable(t *testing.T) {
	first := promotionDocumentID("customer/offer")
	if first != promotionDocumentID("customer/offer") || first == "customer/offer" || len(first) != 64 {
		t.Fatalf("promotionDocumentID returned %q", first)
	}
	if first == promotionDocumentID("customer/other") {
		t.Fatal("different aggregate IDs produced the same document ID")
	}
}

func newTestPromotion(t *testing.T) aggregates.Promotion {
	t.Helper()
	id, err := valueobjects.NewPromotionID("holiday-2026")
	if err != nil {
		t.Fatal(err)
	}
	offer, err := valueobjects.NewOfferReference("offer/winter")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := valueobjects.NewEligibilityPolicyReference("policy/loyal-v2")
	if err != nil {
		t.Fatal(err)
	}
	period, err := valueobjects.NewPeriod(time.Unix(1_800_000_000, 0).UTC(), time.Unix(1_800_086_400, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	value, err := aggregates.NewPromotion(id, offer, policy, period)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
