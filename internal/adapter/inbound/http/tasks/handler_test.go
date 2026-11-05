package tasks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

type fakeVerifier struct{ good bool }

func (v fakeVerifier) Verify(_ context.Context, token string) error {
	if v.good && token == "fake-verified-token" {
		return nil
	}
	return errors.New("invalid token")
}

type fakeStore struct{}

type inFlightTaskStore struct{ fakeStore }

func (inFlightTaskStore) Claim(context.Context, string, string, time.Time) (messagingdelivery.EventJob, bool, error) {
	return messagingdelivery.EventJob{}, false, messagingdelivery.ErrDeliveryInFlight
}

func (fakeStore) Record(context.Context, messagingdelivery.EventJob) (bool, error) { return true, nil }
func (fakeStore) Claim(context.Context, string, string, time.Time) (messagingdelivery.EventJob, bool, error) {
	return messagingdelivery.EventJob{}, false, nil
}
func (fakeStore) Complete(context.Context, string, string, messagingdelivery.DeliveryState) error {
	return nil
}
func (fakeStore) Recover(context.Context, time.Time, int) ([]string, error) { return nil, nil }

type fakeQueue struct{}

func (fakeQueue) Enqueue(context.Context, messagingdelivery.EventKey) error { return nil }

type fakeSender struct{}

func (fakeSender) Reply(context.Context, string, []messagingdelivery.Message) error { return nil }
func (fakeSender) Push(context.Context, string, []messagingdelivery.Message, string) error {
	return nil
}

func TestWorkerRequiresVerifiedOIDC(t *testing.T) {
	store := fakeStore{}
	processor, _ := messagingdelivery.NewProcessor(store, fakeSender{})
	dispatcher, _ := messagingdelivery.NewDispatcher(store, fakeQueue{})
	handler, err := NewHandler(processor, dispatcher, fakeVerifier{good: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	key := strings.Repeat("a", 64)
	for _, tc := range []struct {
		path, token, body string
		want              int
	}{
		{"/tasks/v2/line-reply", "", `{"schema_version":2,"delivery_id":"` + key + `"}`, http.StatusUnauthorized},
		{"/tasks/v2/line-reply", "bad", `{"schema_version":2,"delivery_id":"` + key + `"}`, http.StatusUnauthorized},
		{"/tasks/v2/line-reply", "fake-verified-token", `{"schema_version":2,"delivery_id":"bad"}`, http.StatusBadRequest},
		{"/tasks/v2/line-reply", "fake-verified-token", `{"schema_version":2,"delivery_id":"` + key + `"}`, http.StatusNoContent},
		{"/tasks/v2/dispatch", "fake-verified-token", "{}", http.StatusNoContent},
		{"/tasks/line-reply", "fake-verified-token", `{"key":"` + key + `"}`, http.StatusNoContent},
		{"/tasks/dispatch", "fake-verified-token", "{}", http.StatusNoContent},
	} {
		r := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		if tc.token != "" {
			r.Header.Set("Authorization", "Bearer "+tc.token)
		}
		if strings.HasSuffix(tc.path, "/line-reply") {
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s status %d, want %d", tc.path, w.Code, tc.want)
		}
	}
}

func TestWorkerRetriesActiveDeliveryClaim(t *testing.T) {
	processor, _ := messagingdelivery.NewProcessor(inFlightTaskStore{}, fakeSender{})
	dispatcher, _ := messagingdelivery.NewDispatcher(fakeStore{}, fakeQueue{})
	handler, _ := NewHandler(processor, dispatcher, fakeVerifier{good: true}, nil)
	key := strings.Repeat("a", 64)
	request := httptest.NewRequest(http.MethodPost, "/tasks/v2/line-reply", strings.NewReader(`{"schema_version":2,"delivery_id":"`+key+`"}`))
	request.Header.Set("Authorization", "Bearer fake-verified-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "2" {
		t.Fatalf("in-flight status=%d retry-after=%q", response.Code, response.Header().Get("Retry-After"))
	}
}

func TestLegacyDeliveryTaskRejectsAmbiguousJSON(t *testing.T) {
	key := strings.Repeat("a", 64)
	for _, body := range []string{
		`{"key":"` + key + `","key":"` + key + `"}`,
		`{"key":"` + key + `","extra":true}`,
		`{"delivery_id":"` + key + `"}`,
	} {
		if _, err := decodeLegacyDeliveryTask(strings.NewReader(body)); err == nil {
			t.Fatalf("decodeLegacyDeliveryTask(%q) error = nil", body)
		}
	}
}

func TestDeliveryTaskRejectsAmbiguousJSONAndMediaType(t *testing.T) {
	store := fakeStore{}
	processor, _ := messagingdelivery.NewProcessor(store, fakeSender{})
	dispatcher, _ := messagingdelivery.NewDispatcher(store, fakeQueue{})
	handler, _ := NewHandler(processor, dispatcher, fakeVerifier{good: true}, nil)
	key := strings.Repeat("a", 64)
	for _, tc := range []struct {
		contentType, body string
		want              int
	}{
		{"text/plain", `{"schema_version":2,"delivery_id":"` + key + `"}`, http.StatusUnsupportedMediaType},
		{"application/json", `{"schema_version":2,"delivery_id":"` + key + `","delivery_id":"` + key + `"}`, http.StatusBadRequest},
		{"application/json", `{"schema_version":1,"delivery_id":"` + key + `"}`, http.StatusBadRequest},
		{"application/json", `{"schema_version":2,"delivery_id":"` + key + `","extra":true}`, http.StatusBadRequest},
		{"application/json", `{"schema_version":2,"delivery_id":"` + key + `"} {}`, http.StatusBadRequest},
	} {
		req := httptest.NewRequest(http.MethodPost, "/tasks/v2/line-reply", strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer fake-verified-token")
		req.Header.Set("Content-Type", tc.contentType)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != tc.want {
			t.Fatalf("content=%q body=%q status=%d want=%d", tc.contentType, tc.body, response.Code, tc.want)
		}
	}
}
