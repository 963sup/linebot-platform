// Package aggregates owns Promotion consistency boundaries.
package aggregates

import (
	"errors"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/promotion/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"
)

var (
	ErrInvalidPromotion           = errors.New("invalid promotion")
	ErrInvalidPromotionTransition = errors.New("invalid promotion lifecycle transition")
	ErrPromotionNotActive         = errors.New("promotion is not active")
	ErrPromotionClosed            = errors.New("closed promotion cannot create a delivery intent")
	ErrOutsidePromotionPeriod     = errors.New("time is outside promotion period")
	ErrIneligibleRecipient        = errors.New("recipient is not eligible for promotion")
	ErrEligibilityPolicyMismatch  = errors.New("eligibility decision policy does not match promotion")
)

type Status uint8

const (
	StatusDraft Status = iota + 1
	StatusActive
	StatusClosed
)

func (s Status) Valid() bool { return s >= StatusDraft && s <= StatusClosed }

// Promotion owns one time-bounded offer and its lifecycle.
type Promotion struct {
	id      valueobjects.PromotionID
	offer   valueobjects.OfferReference
	policy  valueobjects.EligibilityPolicyReference
	period  valueobjects.Period
	status  Status
	pending []events.PromotionEvent
}

func NewPromotion(id valueobjects.PromotionID, offer valueobjects.OfferReference, policy valueobjects.EligibilityPolicyReference, period valueobjects.Period) (Promotion, error) {
	if !id.Valid() || !offer.Valid() || !policy.Valid() || !period.Valid() {
		return Promotion{}, ErrInvalidPromotion
	}
	return Promotion{
		id: id, offer: offer, policy: policy, period: period, status: StatusDraft,
		pending: []events.PromotionEvent{events.Defined{ID: id}},
	}, nil
}

// RestorePromotion rehydrates persisted state without raising events.
func RestorePromotion(id valueobjects.PromotionID, offer valueobjects.OfferReference, policy valueobjects.EligibilityPolicyReference, period valueobjects.Period, status Status) (Promotion, error) {
	if !id.Valid() || !offer.Valid() || !policy.Valid() || !period.Valid() || !status.Valid() {
		return Promotion{}, ErrInvalidPromotion
	}
	return Promotion{id: id, offer: offer, policy: policy, period: period, status: status}, nil
}

func (p Promotion) ID() valueobjects.PromotionID                               { return p.id }
func (p Promotion) Offer() valueobjects.OfferReference                         { return p.offer }
func (p Promotion) EligibilityPolicy() valueobjects.EligibilityPolicyReference { return p.policy }
func (p Promotion) Period() valueobjects.Period                                { return p.period }
func (p Promotion) Status() Status                                             { return p.status }
func (p Promotion) PendingEvents() []events.PromotionEvent {
	return append([]events.PromotionEvent(nil), p.pending...)
}

func (p *Promotion) Activate(at time.Time) error {
	if p == nil || p.status != StatusDraft {
		return ErrInvalidPromotionTransition
	}
	if !p.period.Contains(at) {
		return ErrOutsidePromotionPeriod
	}
	p.status = StatusActive
	p.pending = append(p.pending, events.Activated{ID: p.id})
	return nil
}

func (p *Promotion) Close() error {
	if p == nil || p.status != StatusActive {
		return ErrInvalidPromotionTransition
	}
	p.status = StatusClosed
	p.pending = append(p.pending, events.Closed{ID: p.id})
	return nil
}

// CreateDeliveryIntent creates an approved, provider-independent intent. A
// closed promotion can never issue another intent.
func (p *Promotion) CreateDeliveryIntent(recipient string, decision valueobjects.EligibilityDecision, at time.Time) (valueobjects.DeliveryIntent, error) {
	if p == nil {
		return valueobjects.DeliveryIntent{}, ErrInvalidPromotion
	}
	if p.status == StatusClosed {
		return valueobjects.DeliveryIntent{}, ErrPromotionClosed
	}
	if p.status != StatusActive {
		return valueobjects.DeliveryIntent{}, ErrPromotionNotActive
	}
	if !p.period.Contains(at) {
		return valueobjects.DeliveryIntent{}, ErrOutsidePromotionPeriod
	}
	if !decision.Valid() || decision.Policy() != p.policy {
		return valueobjects.DeliveryIntent{}, ErrEligibilityPolicyMismatch
	}
	if !decision.Eligible() {
		return valueobjects.DeliveryIntent{}, ErrIneligibleRecipient
	}
	intent, err := valueobjects.NewDeliveryIntent(p.id, recipient, p.offer)
	if err != nil {
		return valueobjects.DeliveryIntent{}, err
	}
	p.pending = append(p.pending, events.DeliveryIntentCreated{Intent: intent})
	return intent, nil
}
