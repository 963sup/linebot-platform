package requestvalidation

import (
	"context"
	"strings"
	"testing"
)

type validatorStub struct{ calls int }

func (s *validatorStub) Validate(_ context.Context, request Request) (Report, error) {
	s.calls++
	return Report{Valid: len(request.Messages) > 0}, nil
}

func TestValidateDispatchesTypedRequest(t *testing.T) {
	stub := &validatorStub{}
	service, err := New(stub)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Delivery: Reply, Messages: []Message{{Kind: Text, Text: "hello"}, {Kind: Flex, Flex: FlexCard{AltText: "card", Title: "title", Body: "body"}}}}
	report, err := service.Validate(context.Background(), request)
	if err != nil || !report.Valid || stub.calls != 1 {
		t.Fatalf("report=%+v err=%v calls=%d", report, err, stub.calls)
	}
}

func TestInvalidRequestNeverReachesGateway(t *testing.T) {
	stub := &validatorStub{}
	service, _ := New(stub)
	invalid := []Request{
		{},
		{Delivery: "unknown", Messages: []Message{{Kind: Text, Text: "hello"}}},
		{Delivery: Push, Messages: []Message{{Kind: Text, Text: " "}}},
		{Delivery: Push, Messages: []Message{{Kind: Text, Text: strings.Repeat("x", 5001)}}},
		{Delivery: Broadcast, Messages: []Message{{Kind: Flex, Flex: FlexCard{AltText: "card"}}}},
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
