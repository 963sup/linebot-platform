package contactdirectory

import (
	"context"
	"errors"
	"testing"
)

type fakeReader struct {
	profile ProfileQuery
	err     error
}

func (f *fakeReader) GetProfile(_ context.Context, query ProfileQuery) (ProfileSnapshot, error) {
	f.profile = query
	return ProfileSnapshot{Recipient: query.Recipient, DisplayName: "Ada"}, f.err
}

func TestQueriesDispatchTypedContactReads(t *testing.T) {
	reader := &fakeReader{}
	queries, err := NewQueries(reader)
	if err != nil {
		t.Fatal(err)
	}
	profile := ProfileQuery{Recipient: RecipientReference{ID: "group-1", Kind: RecipientGroup}}
	if got, err := queries.GetProfile(context.Background(), profile); err != nil || got.DisplayName != "Ada" || reader.profile != profile {
		t.Fatalf("GetProfile() = %+v, %v", got, err)
	}
}

func TestQueriesRejectInvalidContactReadsAndPropagateErrors(t *testing.T) {
	if _, err := NewQueries(nil); err == nil {
		t.Fatal("NewQueries(nil) succeeded")
	}
	portErr := errors.New("reader failed")
	queries, _ := NewQueries(&fakeReader{err: portErr})
	if _, err := queries.GetProfile(context.Background(), ProfileQuery{Recipient: RecipientReference{ID: "u", Kind: RecipientUser}}); !errors.Is(err, portErr) {
		t.Fatalf("GetProfile error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, call := range map[string]func() error{
		"nil context": func() error {
			_, err := queries.GetProfile(nil, ProfileQuery{Recipient: RecipientReference{ID: "u", Kind: RecipientUser}})
			return err
		},
		"cancelled": func() error {
			_, err := queries.GetProfile(ctx, ProfileQuery{Recipient: RecipientReference{ID: "u", Kind: RecipientUser}})
			return err
		},
		"empty ID": func() error {
			_, err := queries.GetProfile(context.Background(), ProfileQuery{Recipient: RecipientReference{Kind: RecipientUser}})
			return err
		},
		"spaced ID": func() error {
			_, err := queries.GetProfile(context.Background(), ProfileQuery{Recipient: RecipientReference{ID: " u", Kind: RecipientUser}})
			return err
		},
		"bad kind": func() error {
			_, err := queries.GetProfile(context.Background(), ProfileQuery{Recipient: RecipientReference{ID: "u", Kind: "channel"}})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
