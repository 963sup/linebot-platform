// Package valueobjects defines immutable Promotion values.
package valueobjects

import (
	"errors"
	"strings"
	"time"
)

type PromotionID string

func NewPromotionID(raw string) (PromotionID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("promotion id is required")
	}
	return PromotionID(raw), nil
}

func (id PromotionID) String() string { return string(id) }
func (id PromotionID) Valid() bool    { return strings.TrimSpace(string(id)) != "" }

type OfferReference string

func NewOfferReference(raw string) (OfferReference, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("offer reference is required")
	}
	return OfferReference(raw), nil
}

func (r OfferReference) String() string { return string(r) }
func (r OfferReference) Valid() bool    { return strings.TrimSpace(string(r)) != "" }

type EligibilityPolicyReference string

func NewEligibilityPolicyReference(raw string) (EligibilityPolicyReference, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("eligibility policy reference is required")
	}
	return EligibilityPolicyReference(raw), nil
}

func (r EligibilityPolicyReference) String() string { return string(r) }
func (r EligibilityPolicyReference) Valid() bool    { return strings.TrimSpace(string(r)) != "" }

// Period is a half-open promotion interval [start, end).
type Period struct {
	start time.Time
	end   time.Time
}

func NewPeriod(start, end time.Time) (Period, error) {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return Period{}, errors.New("promotion period requires start before end")
	}
	return Period{start: start, end: end}, nil
}

func (p Period) Start() time.Time { return p.start }
func (p Period) End() time.Time   { return p.end }
func (p Period) Valid() bool      { return !p.start.IsZero() && p.end.After(p.start) }
func (p Period) Contains(at time.Time) bool {
	return p.Valid() && !at.Before(p.start) && at.Before(p.end)
}

// EligibilityDecision is an explicit decision made under one policy version.
type EligibilityDecision struct {
	policy   EligibilityPolicyReference
	eligible bool
}

func NewEligibilityDecision(policy EligibilityPolicyReference, eligible bool) (EligibilityDecision, error) {
	if !policy.Valid() {
		return EligibilityDecision{}, errors.New("eligibility decision requires a policy reference")
	}
	return EligibilityDecision{policy: policy, eligible: eligible}, nil
}

func (d EligibilityDecision) Policy() EligibilityPolicyReference { return d.policy }
func (d EligibilityDecision) Eligible() bool                     { return d.eligible }
func (d EligibilityDecision) Valid() bool                        { return d.policy.Valid() }

// DeliveryIntent is an approved request for Messaging Delivery. It contains
// references only and does not expose provider request types.
type DeliveryIntent struct {
	promotion PromotionID
	recipient string
	offer     OfferReference
}

func NewDeliveryIntent(promotion PromotionID, recipient string, offer OfferReference) (DeliveryIntent, error) {
	recipient = strings.TrimSpace(recipient)
	if !promotion.Valid() || recipient == "" || !offer.Valid() {
		return DeliveryIntent{}, errors.New("delivery intent requires promotion recipient and offer")
	}
	return DeliveryIntent{promotion: promotion, recipient: recipient, offer: offer}, nil
}

func (i DeliveryIntent) PromotionID() PromotionID       { return i.promotion }
func (i DeliveryIntent) RecipientReference() string     { return i.recipient }
func (i DeliveryIntent) OfferReference() OfferReference { return i.offer }
func (i DeliveryIntent) Valid() bool {
	return i.promotion.Valid() && i.recipient != "" && i.offer.Valid()
}
