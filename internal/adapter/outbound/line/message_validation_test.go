package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	requestvalidation "github.com/963sup/linebot-platform/internal/application/usecases/request-validation"
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
	message := []requestvalidation.Message{{Kind: requestvalidation.Text, Text: "hello"}}
	for _, delivery := range []requestvalidation.DeliveryKind{requestvalidation.Reply, requestvalidation.Push, requestvalidation.Multicast, requestvalidation.Narrowcast, requestvalidation.Broadcast} {
		call := func() error {
			_, err := s.Validate(ctx, requestvalidation.Request{Delivery: delivery, Messages: message})
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
