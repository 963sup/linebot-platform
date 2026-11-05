package cli

import (
	"bytes"
	"testing"
)

func TestAudienceQueriesFailClosed(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "list"},
		{"--read", "--kind", "group"},
		{"--read", "--kind", "shared-list", "--size", "10000"},
		{"--read", "--kind", "nonexistent"},
	} {
		if err := runAudiences(testContext(), args, getenv, &output); err == nil {
			t.Fatalf("invalid audience inspection accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("invalid audience query should not output data")
	}
}
