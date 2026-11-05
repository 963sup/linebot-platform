package line

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

type fakeMessaging struct {
	replies []messagingdelivery.Message
}

func (f *fakeMessaging) Reply(_ context.Context, _ string, messages []messagingdelivery.Message) error {
	f.replies = append(f.replies, messages...)
	return nil
}

func (f *fakeMessaging) Push(_ context.Context, _ string, _ []messagingdelivery.Message, _ string) error {
	return nil
}

func (f *fakeMessaging) Broadcast(_ context.Context, _ []messagingdelivery.Message, _ string) error {
	return nil
}

func (f *fakeMessaging) Multicast(_ context.Context, _ []string, _ []messagingdelivery.Message, _ string) error {
	return nil
}

func handler(t *testing.T, secret string) (*Handler, *fakeMessaging) {
	t.Helper()
	mock := &fakeMessaging{}
	svc, err := messagingdelivery.NewDeliveryService(mock)
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(secret, svc, nil), mock
}

func signedRequest(secret, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/webhook/line", strings.NewReader(body))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	r.Header.Set("X-Line-Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestSignatureAndMethod(t *testing.T) {
	h, _ := handler(t, "test-secret")
	for _, tt := range []struct {
		name string
		req  *http.Request
		code int
	}{
		{"method", httptest.NewRequest(http.MethodGet, "/webhook/line", nil), http.StatusMethodNotAllowed},
		{"no signature", httptest.NewRequest(http.MethodPost, "/webhook/line", strings.NewReader("{}")), http.StatusUnauthorized},
		{"oversize", signedRequest("test-secret", strings.Repeat("x", maxBodyBytes+1)), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, tt.req)
			if w.Code != tt.code {
				t.Fatalf("code=%d want=%d", w.Code, tt.code)
			}
		})
	}
}

func TestSignedTextAndFlex(t *testing.T) {
	h, mock := handler(t, "test-secret")
	for _, message := range []string{"flex", "hello"} {
		body := `{"destination":"Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","events":[{"type":"message","mode":"active","timestamp":1690000000000,"webhookEventId":"01H00ABCDEF0123456789ABCDE","deliveryContext":{"isRedelivery":false},"source":{"type":"user","userId":"Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},"replyToken":"test-reply-token","message":{"id":"1234567890123","type":"text","text":"` + message + `"}}]}`
		w := httptest.NewRecorder()
		h.ServeHTTP(w, signedRequest("test-secret", body))
		if w.Code != http.StatusOK {
			t.Fatalf("message %q status = %d body=%s", message, w.Code, w.Body.String())
		}
	}
	if len(mock.replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(mock.replies))
	}
	if mock.replies[0].Kind != messagingdelivery.KindFlex || mock.replies[1].Kind != messagingdelivery.KindText {
		t.Fatalf("incorrect message types %#v", mock.replies)
	}
}
