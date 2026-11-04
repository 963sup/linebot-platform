package contacts

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeProfileReader struct {
	calls int
	id    string
	value outbound.UserProfile
	err   error
}

func (f *fakeProfileReader) GetProfile(_ context.Context, userID string) (outbound.UserProfile, error) {
	f.calls++
	f.id = userID
	return f.value, f.err
}

func TestProfilesGetAndValidate(t *testing.T) {
	if _, err := NewProfiles(nil); err == nil {
		t.Fatal("nil profile reader must fail")
	}
	reader := &fakeProfileReader{value: outbound.UserProfile{ID: "U123", DisplayName: "Example"}}
	profiles, err := NewProfiles(reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		ctx  context.Context
		id   string
	}{
		{name: "nil context", id: "U123"},
		{name: "empty ID", ctx: context.Background()},
		{name: "whitespace ID", ctx: context.Background(), id: " U123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := profiles.Get(tc.ctx, tc.id); err == nil {
				t.Fatal("invalid lookup must fail")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := profiles.Get(ctx, "U123"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if reader.calls != 0 {
		t.Fatalf("invalid inputs reached reader %d times", reader.calls)
	}
	result, err := profiles.Get(context.Background(), "U123")
	if err != nil || result.DisplayName != "Example" || reader.id != "U123" || reader.calls != 1 {
		t.Fatalf("unexpected profile result: %#v, %v", result, err)
	}
	reader.err = errors.New("provider failed")
	if _, err := profiles.Get(context.Background(), "U123"); !errors.Is(err, reader.err) {
		t.Fatalf("provider error not propagated: %v", err)
	}
}
