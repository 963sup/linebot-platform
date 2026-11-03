package cli

import (
	"bytes"
	"testing"
)

func TestContactsFailClosed(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "followers"},
		{"--read", "--kind", "unknown"},
		{"--read", "--kind", "group-member-profile", "--conversation-id", "C123"},
		{"--read", "--kind", "group-summary"},
	} {
		if err := runContacts(args, getenv, &output); err == nil {
			t.Fatalf("invalid contacts inspection accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("rejected contacts queries must not expose data")
	}
}
