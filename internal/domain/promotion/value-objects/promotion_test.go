package valueobjects_test

import (
	"testing"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
)

func TestPeriodIsHalfOpen(t *testing.T) {
	t.Parallel()

	start := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	period, err := valueobjects.NewPeriod(start, end)
	if err != nil {
		t.Fatalf("NewPeriod(%v, %v) error = %v, want nil", start, end, err)
	}
	if !period.Contains(start) {
		t.Errorf("Period.Contains(%v) = false, want true", start)
	}
	if period.Contains(end) {
		t.Errorf("Period.Contains(%v) = true, want false", end)
	}
}

func TestNewPeriodRejectsInvalidRange(t *testing.T) {
	t.Parallel()

	now := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, err := valueobjects.NewPeriod(now, now); err == nil {
		t.Errorf("NewPeriod(%v, %v) error = nil, want non-nil", now, now)
	}
}

func TestNewDeliveryIntentRequiresRecipient(t *testing.T) {
	t.Parallel()

	id, err := valueobjects.NewPromotionID("promotion")
	if err != nil {
		t.Fatalf("NewPromotionID(...) error = %v, want nil", err)
	}
	offer, err := valueobjects.NewOfferReference("offer")
	if err != nil {
		t.Fatalf("NewOfferReference(...) error = %v, want nil", err)
	}
	if _, err := valueobjects.NewDeliveryIntent(id, "", offer); err == nil {
		t.Error("NewDeliveryIntent(..., empty recipient, ...) error = nil, want non-nil")
	}
}
