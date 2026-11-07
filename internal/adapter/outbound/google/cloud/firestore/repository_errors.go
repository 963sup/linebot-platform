package firestore

import (
	"context"
	"errors"
	"fmt"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/domain/messaging-delivery/aggregates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrRepositoryCorrupt reports persisted state that cannot be rehydrated.
	ErrRepositoryCorrupt = errors.New("repository aggregate state is corrupt")
	// ErrRepositoryUnavailable reports an infrastructure failure while accessing persistence.
	ErrRepositoryUnavailable = errors.New("repository infrastructure unavailable")
)

// classifyMessagingStoreError prevents Firestore transport details from
// crossing the messaging-delivery port. Domain retry sentinels and context
// cancellation retain their control-flow identity.
func classifyMessagingStoreError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, messagingdelivery.ErrDeliveryInFlight) {
		return messagingdelivery.ErrDeliveryInFlight
	}
	for _, sentinel := range []error{
		aggregates.ErrNotClaimable,
		aggregates.ErrClaimMismatch,
		aggregates.ErrTerminal,
		ErrRepositoryCorrupt,
		ErrRepositoryUnavailable,
	} {
		if errors.Is(err, sentinel) {
			return err
		}
	}
	if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
		return context.DeadlineExceeded
	}
	if _, ok := status.FromError(err); ok {
		return fmt.Errorf("%w: messaging delivery %s", ErrRepositoryUnavailable, operation)
	}
	return fmt.Errorf("%w: messaging delivery %s", ErrRepositoryCorrupt, operation)
}
