package cli

import (
	"bytes"
	"testing"
)

func TestInsightsFailClosed(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "demographics"},
		{"--read", "--kind", "unknown"},
		{"--read", "--kind", "deliveries", "--date", "20261334"},
		{"--read", "--kind", "menu-daily", "--rich-menu", "rm", "--from", "20261103"},
	} {
		if err := runInsights(args, getenv, &output); err == nil {
			t.Fatalf("invalid insights args accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("invalid insights lookup must not expose data")
	}
}
