package tasks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/ingestion"
	"time"
)

type fakeVerifier struct{ good bool }

func (v fakeVerifier) Verify(_ context.Context, token string) error {
	if v.good && token == "fake-verified-token" {
		return nil
	}
	return errors.New("invalid token")
}

type fakeStore struct{}
func (fakeStore) Record(context.Context, outbound.EventJob) (bool, error) { return true, nil }
func (fakeStore) Claim(context.Context, string, string, time.Time) (outbound.EventJob, bool, error) { return outbound.EventJob{}, false, nil }
func (fakeStore) Complete(context.Context, string, string, outbound.DeliveryState) error { return nil }
func (fakeStore) Recover(context.Context, time.Time, int) ([]string, error) { return nil, nil }

type fakeQueue struct{}
func (fakeQueue) Enqueue(context.Context, string) error { return nil }

type fakeSender struct{}
func (fakeSender) Reply(context.Context, string, []outbound.Message) error { return nil }
func (fakeSender) Push(context.Context, string, []outbound.Message, string) error { return nil }

func TestWorkerRequiresVerifiedOIDC(t *testing.T) {
	store := fakeStore{}
	processor, _ := ingestion.NewProcessor(store, fakeSender{})
	dispatcher, _ := ingestion.NewDispatcher(store, fakeQueue{})
	handler, err := NewHandler(processor, dispatcher, fakeVerifier{good: true}, nil)
	if err != nil { t.Fatal(err) }
	key := strings.Repeat("a", 64)
	for _, tc := range []struct {
		path, token, body string
		want int
	}{
		{"/tasks/line-reply", "", `{"key":"`+key+`"}`, http.StatusUnauthorized},
		{"/tasks/line-reply", "bad", `{"key":"`+key+`"}`, http.StatusUnauthorized},
		{"/tasks/line-reply", "fake-verified-token", `{"key":"bad"}`, http.StatusBadRequest},
		{"/tasks/line-reply", "fake-verified-token", `{"key":"`+key+`"}`, http.StatusNoContent},
		{"/tasks/dispatch", "fake-verified-token", "{}", http.StatusNoContent},
	} {
		r := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		if tc.token != "" { r.Header.Set("Authorization", "Bearer "+tc.token) }
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s status %d, want %d", tc.path, w.Code, tc.want)
		}
	}
}
