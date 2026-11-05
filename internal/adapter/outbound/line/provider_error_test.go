package line

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
)

func TestLINEProviderErrorRemovesRawSDKDetailsFromEntireChain(t *testing.T) {
	const (
		secretURL     = "https://api.line.example/v2/bot/message?access_token=private-token"
		secretBody    = `{"message":"raw provider payload","userId":"U-private-user"}`
		secretToken   = "private-token"
		secretUserID  = "U-private-user"
		secretPayload = "raw provider payload"
	)
	raw := &url.Error{Op: "POST", URL: secretURL, Err: fmt.Errorf("response body %s", secretBody)}
	body := &providerErrorCloseTracker{Reader: strings.NewReader(secretBody)}
	err := lineAPIError("push", &http.Response{StatusCode: http.StatusBadGateway, Body: body}, raw)
	if err == nil {
		t.Fatal("lineAPIError() error = nil")
	}
	if !body.closed {
		t.Fatal("LINE response body was not closed")
	}
	assertSafeLINEErrorChain(t, err, secretURL, secretBody, secretToken, secretUserID, secretPayload)

	var external *resilience.ExternalError
	if !errors.As(err, &external) {
		t.Fatalf("lineAPIError() type = %T, want *resilience.ExternalError", err)
	}
	if external.StatusCode != http.StatusBadGateway || external.Kind != resilience.KindUnavailable || external.OutcomeUnknown {
		t.Fatalf("external error = %#v", external)
	}
}

func TestLINEProviderErrorPreservesContextAndUnknownOutcomeSemantics(t *testing.T) {
	tests := []struct {
		name     string
		raw      error
		want     error
		wantKind resilience.Kind
	}{
		{
			name:     "canceled",
			raw:      fmt.Errorf("request https://secret.invalid?token=private: %w", context.Canceled),
			want:     context.Canceled,
			wantKind: resilience.KindCanceled,
		},
		{
			name:     "deadline",
			raw:      fmt.Errorf("provider body and user U-secret: %w", context.DeadlineExceeded),
			want:     context.DeadlineExceeded,
			wantKind: resilience.KindTimeout,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := lineAPIError("reply", nil, test.raw)
			if !errors.Is(err, test.want) {
				t.Fatalf("lineAPIError() = %v, want errors.Is(_, %v)", err, test.want)
			}
			var external *resilience.ExternalError
			if !errors.As(err, &external) || external.Kind != test.wantKind || !external.OutcomeUnknown || external.StatusCode != 0 {
				t.Fatalf("external error = %#v", external)
			}
			assertSafeLINEErrorChain(t, err, "secret.invalid", "private", "provider body", "U-secret")
		})
	}
}

func assertSafeLINEErrorChain(t *testing.T, err error, forbidden ...string) {
	t.Helper()
	for current := err; current != nil; current = errors.Unwrap(current) {
		message := current.Error()
		for _, value := range forbidden {
			if value != "" && strings.Contains(message, value) {
				t.Fatalf("unsafe provider detail %q leaked through error %T: %q", value, current, message)
			}
		}
	}
}

type providerErrorCloseTracker struct {
	io.Reader
	closed bool
}

func (b *providerErrorCloseTracker) Close() error {
	b.closed = true
	return nil
}
