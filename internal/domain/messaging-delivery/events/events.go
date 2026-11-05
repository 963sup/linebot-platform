package events

import (
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

// DeliveryEvent marks facts raised by the Delivery aggregate.
type DeliveryEvent interface{ DeliveryEvent() }

type Claimed struct {
	DeliveryID valueobjects.DeliveryID
	At         time.Time
}

func (Claimed) DeliveryEvent() {}

type Completed struct {
	DeliveryID valueobjects.DeliveryID
	Outcome    valueobjects.Status
	At         time.Time
}

func (Completed) DeliveryEvent() {}

type OutcomeUnknown struct {
	DeliveryID valueobjects.DeliveryID
	At         time.Time
}

func (OutcomeUnknown) DeliveryEvent() {}
