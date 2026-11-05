package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
)

type blockingProfileRuntime struct {
	entered chan context.Context
}

func (r *blockingProfileRuntime) GetProfile(ctx context.Context, _ contactdirectory.ProfileQuery) (contactdirectory.ProfileSnapshot, error) {
	r.entered <- ctx
	<-ctx.Done()
	return contactdirectory.ProfileSnapshot{}, ctx.Err()
}
func (*blockingProfileRuntime) Close() error { return nil }

func TestProfileLookupRequiresExplicitFlagAndCredentials(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--user", "U123"},
		{"--lookup"},
		{"--lookup", "--user", " U123"},
		{"--lookup", "--user", "U123", "extra"},
	} {
		if err := runProfile(testContext(), args, getenv, &output); err == nil {
			t.Fatalf("invalid profile args unexpectedly accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("invalid lookup must not write profile data")
	}
	err := runProfile(testContext(), []string{"--lookup", "--user", "U123"}, getenv, &output)
	if err == nil || !strings.Contains(err.Error(), "access token") {
		t.Fatalf("missing token must fail before network call: %v", err)
	}
	if output.Len() != 0 {
		t.Fatal("failed lookup must not write profile data")
	}
}

func TestProfilePropagatesParentContextDuringExecution(t *testing.T) {
	type contextKey struct{}
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "profile"))
	runtime := &blockingProfileRuntime{entered: make(chan context.Context, 1)}
	deps := Dependencies{ProfileRuntime: func(string) (ProfileRuntime, error) { return runtime, nil }}
	done := make(chan error, 1)
	go func() {
		done <- runProfile(parent, []string{"--lookup", "--user", "U123"}, func(string) string { return "token" }, &bytes.Buffer{}, deps)
	}()
	ctx := <-runtime.entered
	if ctx.Value(contextKey{}) != "profile" {
		t.Fatal("profile context value was not propagated")
	}
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 12*time.Second {
		t.Fatalf("profile deadline = %v", deadline)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("profile cancellation error = %v", err)
	}
}
