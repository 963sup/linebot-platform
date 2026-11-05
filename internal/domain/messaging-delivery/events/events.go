package events

import (
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

type Claimed struct {
	DeliveryID valueobjects.DeliveryID
	At         time.Time
}

type Completed struct {
	DeliveryID valueobjects.DeliveryID
	Outcome    valueobjects.Status
	At         time.Time
}

type OutcomeUnknown struct {
	DeliveryID valueobjects.DeliveryID
	At         time.Time
}
