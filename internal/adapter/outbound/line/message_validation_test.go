package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLINEMessageValidationEndpoints(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("unexpected LINE validation method or credentials")
		}
		calls = append(calls,r.URL.Path)
		w.Header().Set("Content-Type","application/json")
		_,_ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t,server)
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { return s.ValidateReply(ctx,testText()) },
		func() error { return s.ValidatePush(ctx,testText()) },
		func() error { return s.ValidateMulticast(ctx,testText()) },
		func() error { return s.ValidateNarrowcast(ctx,testText()) },
		func() error { return s.ValidateBroadcast(ctx,testText()) },
	} {
		if err := call(); err != nil { t.Fatal(err) }
	}
	if len(calls) != 5 { t.Fatalf("expected five message validators, got %v",calls) }
}
