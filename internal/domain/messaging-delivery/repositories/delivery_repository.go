package repositories

import (
	"context"

	"github.com/963sup/linebot-platform/internal/domain/messaging-delivery/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
)

// DeliveryRepository persists Delivery with optimistic version enforcement.
type DeliveryRepository interface {
	Find(context.Context, valueobjects.DeliveryID) (aggregates.Delivery, error)
	Save(context.Context, aggregates.Delivery, uint64) error
}
