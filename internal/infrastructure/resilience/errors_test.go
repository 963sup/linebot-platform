package resilience

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestClassifyAndSanitize(t *testing.T) {
	cases := []struct {
		status int
		cause  error
		want   Kind
	}{
		{0, context.Canceled, KindCanceled},
		{0, context.DeadlineExceeded, KindTimeout},
		{401, errors.New("secret token"), KindUnauthorized},
		{403, errors.New("secret token"), KindUnauthorized},
		{429, errors.New("secret token"), KindRateLimited},
		{503, errors.New("secret token"), KindUnavailable},
		{400, errors.New("secret token"), KindUnknown},
	}
	for _, tc := range cases {
		err := Wrap("line", "push", tc.status, tc.cause)
		var external *ExternalError
		if !errors.As(err, &external) || external.Kind != tc.want {
			t.Fatalf("status %d error = %#v, want %s", tc.status, err, tc.want)
		}
		if !errors.Is(err, tc.cause) {
			t.Fatal("original error should be available by unwrapping")
		}
		if external.OutcomeUnknown != (tc.status == 0) {
			t.Fatal("unknown outcome mismatch")
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe error message %q", err.Error())
		}
	}
	if Wrap("line", "push", 200, nil) != nil {
		t.Fatal("nil cause should not be wrapped")
	}
}
