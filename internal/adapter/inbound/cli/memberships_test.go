package cli

import (
	"bytes"
	"testing"
)

func TestMembershipQueriesFailClosed(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "memberships"},
		{"--read", "--kind", "membership-users", "--membership-id", "-1"},
		{"--read", "--kind", "coupon"},
		{"--read", "--kind", "unknown"},
	} {
		if err := runMemberships(testContext(), args, getenv, &output); err == nil {
			t.Fatalf("invalid membership query accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("rejected queries must not expose data")
	}
}
