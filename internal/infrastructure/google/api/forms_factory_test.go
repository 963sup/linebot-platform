package api

import (
	"context"
	"net/http"
	"testing"
)

func TestFormsFactoryUsesAuthorizedTransport(t *testing.T) {
	if _, err := NewForms(context.Background(), nil); err == nil { t.Fatal("nil authorized HTTP client accepted") }
	if _, err := NewForms(nil, &http.Client{}); err == nil { t.Fatal("nil context accepted") }
	svc, err := NewForms(context.Background(), &http.Client{})
	if err != nil || svc == nil || svc.Forms == nil || svc.Forms.Responses == nil || svc.Forms.Watches == nil {
		t.Fatalf("Forms SDK not constructed: %v",err)
	}
}
