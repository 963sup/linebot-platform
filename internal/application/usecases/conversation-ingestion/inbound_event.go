// Package conversationingestion owns provider-neutral inbound conversation contracts.
package conversationingestion

import (
	"errors"
	"strings"
	"time"
)

type AddressKind string

const (
	AddressDirect AddressKind = "direct"
	AddressGroup  AddressKind = "group"
)

type Address struct {
	Kind              AddressKind
	ProviderReference string
	ActorReference    string
}

type InboundTextEvent struct {
	EventID    string
	OccurredAt time.Time
	Address    Address
	Text       string
}

func NewInboundTextEvent(eventID string, occurredAt time.Time, address Address, text string) (InboundTextEvent, error) {
	if strings.TrimSpace(eventID) == "" || occurredAt.IsZero() || strings.TrimSpace(address.ProviderReference) == "" ||
		strings.TrimSpace(text) == "" {
		return InboundTextEvent{}, errors.New("complete inbound conversation event is required")
	}
	switch address.Kind {
	case AddressDirect:
		if address.ActorReference != "" && address.ActorReference != address.ProviderReference {
			return InboundTextEvent{}, errors.New("direct actor must match conversation address")
		}
		address.ActorReference = address.ProviderReference
	case AddressGroup:
	default:
		return InboundTextEvent{}, errors.New("supported conversation address kind is required")
	}
	return InboundTextEvent{EventID: eventID, OccurredAt: occurredAt.UTC(), Address: address, Text: text}, nil
}
