package outbound

import "context"

// MessageValidation performs LINE's read-only request validation for a bounded batch.
type MessageValidation interface {
	ValidateReply(ctx context.Context, messages []Message) error
	ValidatePush(ctx context.Context, messages []Message) error
	ValidateMulticast(ctx context.Context, messages []Message) error
	ValidateNarrowcast(ctx context.Context, messages []Message) error
	ValidateBroadcast(ctx context.Context, messages []Message) error
}
