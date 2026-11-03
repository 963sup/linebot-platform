package messaging

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type ValidationService struct { validator outbound.MessageValidation }

func NewValidationService(validator outbound.MessageValidation) (*ValidationService, error) {
	if validator == nil { return nil, errors.New("message validator is required") }
	return &ValidationService{validator: validator}, nil
}

// Validate checks a typed message batch without sending it to any LINE recipient.
func (s *ValidationService) Validate(ctx context.Context, kind string, messages []outbound.Message) error {
	if ctx == nil { return errors.New("context is required") }
	if err := ctx.Err(); err != nil { return err }
	if len(messages) == 0 || len(messages) > 5 { return errors.New("LINE requires 1-5 messages") }
	for _, message := range messages {
		if err := message.Validate(); err != nil { return err }
	}
	switch kind {
	case "reply":
		return s.validator.ValidateReply(ctx,messages)
	case "push":
		return s.validator.ValidatePush(ctx,messages)
	case "multicast":
		return s.validator.ValidateMulticast(ctx,messages)
	case "narrowcast":
		return s.validator.ValidateNarrowcast(ctx,messages)
	case "broadcast":
		return s.validator.ValidateBroadcast(ctx,messages)
	default:
		return errors.New("unsupported validation kind")
	}
}
