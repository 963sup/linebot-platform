package mediatransfer

import (
	"context"
	"testing"
)

type mediaStub struct{ called string }

func (s *mediaStub) FetchMessageContent(context.Context, string, ContentVariant) (BinaryContent, error) {
	s.called = "message"
	return BinaryContent{Data: []byte("x")}, nil
}
func (s *mediaStub) MessageContentStatus(context.Context, string) (PreparationStatus, error) {
	s.called = "status"
	return PreparationReady, nil
}
func (s *mediaStub) FetchPresentationImage(context.Context, string) (BinaryContent, error) {
	s.called = "image"
	return BinaryContent{Data: []byte("x")}, nil
}
func (s *mediaStub) StorePresentationImage(context.Context, string, BinaryContent) error {
	s.called = "store"
	return nil
}

func TestMediaOperationsAreTypedAndValidated(t *testing.T) {
	stub := &mediaStub{}
	service, err := New(stub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.MessageContent(context.Background(), "message-1", Preview); err != nil || stub.called != "message" {
		t.Fatalf("message: %v", err)
	}
	if status, err := service.MessageContentStatus(context.Background(), "message-1"); err != nil || status != PreparationReady {
		t.Fatalf("status=%q err=%v", status, err)
	}
	if _, err := service.PresentationImage(context.Background(), "presentation-1"); err != nil || stub.called != "image" {
		t.Fatalf("image: %v", err)
	}
	if err := service.StorePresentationImage(context.Background(), "presentation-1", BinaryContent{Data: []byte("png"), ContentType: "image/png"}); err != nil || stub.called != "store" {
		t.Fatalf("store: %v", err)
	}
}

func TestInvalidMediaInputHasNoSideEffect(t *testing.T) {
	stub := &mediaStub{}
	service, _ := New(stub)
	if _, err := service.MessageContent(context.Background(), " message-1", Original); err == nil {
		t.Fatal("invalid ID accepted")
	}
	if _, err := service.MessageContent(context.Background(), "message-1", "raw"); err == nil {
		t.Fatal("invalid variant accepted")
	}
	if err := service.StorePresentationImage(context.Background(), "presentation-1", BinaryContent{Data: []byte("x"), ContentType: "text/plain"}); err == nil {
		t.Fatal("invalid image accepted")
	}
	if stub.called != "" {
		t.Fatalf("invalid input called %q", stub.called)
	}
}
