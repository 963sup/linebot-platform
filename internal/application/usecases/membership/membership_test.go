package membership

import (
	"context"
	"errors"
	"testing"
)

type fakeReader struct {
	listPage       PageRequest
	subscriptionID RecipientID
	membershipID   MembershipID
	memberPage     PageRequest
	err            error
}

func (f *fakeReader) ListMemberships(_ context.Context, page PageRequest) (MembershipPage, error) {
	f.listPage = page
	return MembershipPage{Memberships: []MembershipView{{ID: "plan-1"}}}, f.err
}

func (f *fakeReader) GetSubscription(_ context.Context, id RecipientID) (SubscriptionView, error) {
	f.subscriptionID = id
	return SubscriptionView{RecipientID: id}, f.err
}

func (f *fakeReader) ListMembershipMembers(_ context.Context, id MembershipID, page PageRequest) (MemberPage, error) {
	f.membershipID, f.memberPage = id, page
	return MemberPage{Members: []MemberView{{RecipientID: "user-1"}}}, f.err
}

func TestQueriesDispatchTypedMembershipReads(t *testing.T) {
	reader := &fakeReader{}
	queries, err := NewQueries(reader)
	if err != nil {
		t.Fatal(err)
	}
	page := PageRequest{Token: "next", Limit: 25}
	if got, err := queries.ListMemberships(context.Background(), page); err != nil || len(got.Memberships) != 1 || reader.listPage != page {
		t.Fatalf("ListMemberships() = %+v, %v", got, err)
	}
	if got, err := queries.GetSubscription(context.Background(), "user-1"); err != nil || got.RecipientID != "user-1" || reader.subscriptionID != "user-1" {
		t.Fatalf("GetSubscription() = %+v, %v", got, err)
	}
	if got, err := queries.ListMembers(context.Background(), "plan-1", page); err != nil || len(got.Members) != 1 || reader.membershipID != "plan-1" || reader.memberPage != page {
		t.Fatalf("ListMembers() = %+v, %v", got, err)
	}
}

func TestQueriesNormalizeDefaultPageSize(t *testing.T) {
	reader := &fakeReader{}
	queries, err := NewQueries(reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queries.ListMemberships(context.Background(), PageRequest{}); err != nil || reader.listPage.Limit != DefaultPageSize {
		t.Fatalf("ListMemberships() page = %+v, error = %v", reader.listPage, err)
	}
	if _, err := queries.ListMembers(context.Background(), "plan-1", PageRequest{}); err != nil || reader.memberPage.Limit != DefaultPageSize {
		t.Fatalf("ListMembers() page = %+v, error = %v", reader.memberPage, err)
	}
}

func TestQueriesRejectInvalidInputAndPropagateErrors(t *testing.T) {
	if _, err := NewQueries(nil); err == nil {
		t.Fatal("NewQueries(nil) succeeded")
	}
	portErr := errors.New("reader failed")
	queries, _ := NewQueries(&fakeReader{err: portErr})
	if _, err := queries.ListMemberships(context.Background(), PageRequest{}); !errors.Is(err, portErr) {
		t.Fatalf("ListMemberships error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, call := range map[string]func() error{
		"nil context": func() error { _, err := queries.ListMemberships(nil, PageRequest{}); return err },
		"cancelled":   func() error { _, err := queries.ListMemberships(ctx, PageRequest{}); return err },
		"subscription context": func() error {
			_, err := queries.GetSubscription(nil, "user-1")
			return err
		},
		"members context": func() error {
			_, err := queries.ListMembers(nil, "plan-1", PageRequest{})
			return err
		},
		"bad token": func() error {
			_, err := queries.ListMemberships(context.Background(), PageRequest{Token: " next"})
			return err
		},
		"bad limit": func() error {
			_, err := queries.ListMemberships(context.Background(), PageRequest{Limit: MaxPageSize + 1})
			return err
		},
		"no recipient": func() error {
			_, err := queries.GetSubscription(context.Background(), "")
			return err
		},
		"no membership": func() error {
			_, err := queries.ListMembers(context.Background(), " ", PageRequest{})
			return err
		},
		"bad member page": func() error {
			_, err := queries.ListMembers(context.Background(), "plan-1", PageRequest{Limit: MaxPageSize + 1})
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
