package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCampaignRequiresExplicitSend(t *testing.T) {
	env := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "broadcast", "--text", "hello"},
		{"--kind", "other", "--text", "hello", "--send"},
		{"--kind", "broadcast", "--to", "U123", "--text", "hello", "--send"},
		{"--kind", "multicast", "--text", "hello", "--send"},
		{"--kind", "narrowcast", "--text", "hello", "--send"},
		{"--kind", "narrowcast", "--request", "missing.json", "--send"},
	} {
		if err := runCampaign(testContext(), args, env); err == nil {
			t.Fatalf("unapproved send: %v", args)
		}
	}
}

func TestCampaignNarrowcastRejectsMalformedRequestFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, []byte(`{"messages":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runCampaign(testContext(), []string{"--kind", "narrowcast", "--request", path, "--send"}, func(string) string { return "" }); err == nil {
		t.Fatal("malformed narrowcast request accepted")
	}
}
