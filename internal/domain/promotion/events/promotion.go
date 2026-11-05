// Package events defines facts raised by Promotion.
package events

import valueobjects "github.com/963sup/linebot-platform/internal/domain/promotion/value-objects"

type PromotionEvent interface{ PromotionEvent() }

type Defined struct{ ID valueobjects.PromotionID }

func (Defined) PromotionEvent() {}

type Activated struct{ ID valueobjects.PromotionID }

func (Activated) PromotionEvent() {}

type Closed struct{ ID valueobjects.PromotionID }

func (Closed) PromotionEvent() {}

type DeliveryIntentCreated struct{ Intent valueobjects.DeliveryIntent }

func (DeliveryIntentCreated) PromotionEvent() {}
