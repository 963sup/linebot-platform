package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

func TestLINEMessageValidationEndpoints(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("unexpected LINE validation method or credentials")
		}
		calls = append(calls, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	ctx := context.Background()
	message := []messagingdelivery.Message{{Kind: messagingdelivery.KindText, Text: "hello"}}
	for _, delivery := range []messagingdelivery.DeliveryKind{messagingdelivery.DeliveryReply, messagingdelivery.DeliveryPush, messagingdelivery.DeliveryMulticast, messagingdelivery.DeliveryNarrowcast, messagingdelivery.DeliveryBroadcast} {
		call := func() error {
			_, err := s.Validate(ctx, messagingdelivery.ValidationRequest{Delivery: delivery, Messages: message})
			return err
		}
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 5 {
		t.Fatalf("expected five message validators, got %v", calls)
	}
}

func TestLINEMessageValidationRejectsUnknownDeliveryWithoutProviderCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unknown delivery must not call LINE")
	}))
	defer server.Close()

	_, err := contractSender(t, server).Validate(context.Background(), messagingdelivery.ValidationRequest{
		Delivery: messagingdelivery.DeliveryKind("unknown"),
		Messages: []messagingdelivery.Message{{Kind: messagingdelivery.KindText, Text: "hello"}},
	})
	if err == nil {
		t.Fatal("unknown delivery accepted")
	}
}
