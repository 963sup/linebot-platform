package messagingdelivery

import (
	"context"
	"strings"
	"testing"
)

type validatorStub struct{ calls int }

func (s *validatorStub) Validate(_ context.Context, request ValidationRequest) (ValidationReport, error) {
	s.calls++
	return ValidationReport{Valid: len(request.Messages) > 0}, nil
}

func TestValidationServiceDispatchesCanonicalMessages(t *testing.T) {
	stub := &validatorStub{}
	service, err := NewValidationService(stub)
	if err != nil {
		t.Fatal(err)
	}
	request := ValidationRequest{Delivery: DeliveryReply, Messages: []Message{
		{Kind: KindText, Text: "hello"},
		{Kind: KindFlex, Flex: FlexCard{AltText: "card", Title: "title", Body: "body"}},
	}}
	report, err := service.Validate(context.Background(), request)
	if err != nil || !report.Valid || stub.calls != 1 {
		t.Fatalf("report=%+v err=%v calls=%d", report, err, stub.calls)
	}
}

func TestValidationServiceRejectsInvalidCanonicalMessages(t *testing.T) {
	stub := &validatorStub{}
	service, _ := NewValidationService(stub)
	invalid := []ValidationRequest{
		{},
		{Delivery: "unknown", Messages: []Message{{Kind: KindText, Text: "hello"}}},
		{Delivery: DeliveryPush, Messages: []Message{{Kind: KindText, Text: " "}}},
		{Delivery: DeliveryPush, Messages: []Message{{Kind: KindText, Text: strings.Repeat("x", 5001)}}},
		{Delivery: DeliveryBroadcast, Messages: []Message{{Kind: KindFlex, Flex: FlexCard{AltText: "card"}}}},
	}
	for _, request := range invalid {
		if _, err := service.Validate(context.Background(), request); err == nil {
			t.Fatalf("accepted %+v", request)
		}
	}
	if stub.calls != 0 {
		t.Fatal("invalid request reached gateway")
	}
}
