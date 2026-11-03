package engagement

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeMembershipReader struct {
	users int
	limit int32
}

func (f *fakeMembershipReader) MembershipList(context.Context) (json.RawMessage, error) {
	return json.RawMessage("{}"), nil
}

func (f *fakeMembershipReader) MembershipSubscription(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage("{}"), nil
}

func (f *fakeMembershipReader) MembershipUsers(_ context.Context, _ int32, _ string, limit int32) (json.RawMessage, error) {
	f.users++
	f.limit = limit
	return json.RawMessage("{}"), nil
}

func TestMembershipInspectorRejectsCoupons(t *testing.T) {
	if _, err := NewMembershipInspector(nil); err == nil {
		t.Fatal("nil membership reader must be rejected")
	}
	for _, q := range []MembershipQuery{
		{Kind: "coupon"},
		{Kind: "coupons"},
		{Kind: "membership-users"},
		{Kind: "subscription"},
		{Kind: "membership-users", MembershipID: -1},
	} {
		if err := ValidateMembershipQuery(q); err == nil {
			t.Fatalf("accepted invalid membership query: %+v", q)
		}
	}
	reader := &fakeMembershipReader{}
	inspector, err := NewMembershipInspector(reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inspector.Inspect(context.Background(), MembershipQuery{Kind: "membership-users", MembershipID: 123}); err != nil {
		t.Fatal(err)
	}
	if reader.users != 1 || reader.limit != 20 {
		t.Fatalf("membership users calls=%d limit=%d, want 1,20", reader.users, reader.limit)
	}
}
