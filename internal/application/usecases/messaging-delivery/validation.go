package messagingdelivery

import (
	"context"
	"errors"
)

type DeliveryKind string

const (
	DeliveryReply      DeliveryKind = "reply"
	DeliveryPush       DeliveryKind = "push"
	DeliveryMulticast  DeliveryKind = "multicast"
	DeliveryNarrowcast DeliveryKind = "narrowcast"
	DeliveryBroadcast  DeliveryKind = "broadcast"
)

type ValidationRequest struct {
	Delivery DeliveryKind
	Messages []Message
}

type Violation struct{ Field, Reason string }

type ValidationReport struct {
	Valid      bool
	Violations []Violation
}

type ValidationGateway interface {
	Validate(context.Context, ValidationRequest) (ValidationReport, error)
}

type ValidationService struct{ gateway ValidationGateway }

func NewValidationService(gateway ValidationGateway) (*ValidationService, error) {
	if gateway == nil {
		return nil, errors.New("request validation gateway is required")
	}
	return &ValidationService{gateway: gateway}, nil
}

func (s *ValidationService) Validate(ctx context.Context, request ValidationRequest) (ValidationReport, error) {
	if ctx == nil {
		return ValidationReport{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return ValidationReport{}, err
	}
	if err := ValidateRequest(request); err != nil {
		return ValidationReport{}, err
	}
	request.Messages = append([]Message(nil), request.Messages...)
	return s.gateway.Validate(ctx, request)
}

func ValidateRequest(request ValidationRequest) error {
	switch request.Delivery {
	case DeliveryReply, DeliveryPush, DeliveryMulticast, DeliveryNarrowcast, DeliveryBroadcast:
	default:
		return errors.New("unsupported delivery kind")
	}
	if len(request.Messages) == 0 || len(request.Messages) > 5 {
		return errors.New("request requires 1-5 messages")
	}
	for _, message := range request.Messages {
		if err := message.Validate(); err != nil {
			return err
		}
	}
	return nil
}
