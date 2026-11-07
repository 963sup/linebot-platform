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
	AddressRoom   AddressKind = "room"
)

type ChannelScope struct {
	TenantID, ConnectionID string
}

type Address struct {
	Kind              AddressKind
	ProviderReference string
	ActorReference    string
}

type InboundTextEvent struct {
	Scope      ChannelScope
	EventID    string
	OccurredAt time.Time
	Address    Address
	Text       string
}

func NewInboundTextEvent(scope ChannelScope, eventID string, occurredAt time.Time, address Address, text string) (InboundTextEvent, error) {
	if strings.TrimSpace(scope.TenantID) == "" || strings.TrimSpace(scope.ConnectionID) == "" ||
		strings.TrimSpace(eventID) == "" || occurredAt.IsZero() || strings.TrimSpace(address.ProviderReference) == "" ||
		strings.TrimSpace(text) == "" {
		return InboundTextEvent{}, errors.New("complete inbound conversation event is required")
	}
	switch address.Kind {
	case AddressDirect:
		if address.ActorReference != "" && address.ActorReference != address.ProviderReference {
			return InboundTextEvent{}, errors.New("direct actor must match conversation address")
		}
		address.ActorReference = address.ProviderReference
	case AddressGroup, AddressRoom:
	default:
		return InboundTextEvent{}, errors.New("supported conversation address kind is required")
	}
	return InboundTextEvent{Scope: scope, EventID: eventID, OccurredAt: occurredAt.UTC(), Address: address, Text: text}, nil
}
