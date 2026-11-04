package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenOperationsFailClosed(t *testing.T) {
	for _, args := range [][]string{
		{"--action", "issue-short"},
		{"--action", "issue-short", "--request", "unknown", "--apply", "--confirm", "issue-short", "--out", "token.json"},
		{"--action", "revoke-short", "--request", "unknown", "--read"},
		{"--action", "verify-jwt", "--request", "unknown", "--read"},
	} {
		if err := runChannelTokens(args); err == nil {
			t.Fatalf("invalid token operation accepted: %v", args)
		}
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(p, []byte(`{"accessToken":"secret"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runChannelTokens([]string{"--action", "verify-jwt", "--request", p, "--read", "--out", filepath.Join(dir, "result.json")}); err == nil {
		t.Fatal("world-readable secrets allowed")
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tokenRequestFromPrivateFile(p); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(other, []byte(`{"accessToken":"secret","unknown":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tokenRequestFromPrivateFile(other); err == nil {
		t.Fatal("unknown token request field accepted")
	}
}

func TestPrivateOperationRunsOnlyWithReservedOutput(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "private.json")
	calls := 0
	fn := func() ([]byte, error) { calls++; return []byte("secret"), nil }
	if err := writePrivateOperation(dest, fn); err != nil {
		t.Fatal(err)
	}
	if err := writePrivateOperation(dest, fn); err == nil {
		t.Fatal("existing secret output overwritten")
	}
	if calls != 1 {
		t.Fatalf("external operation was called %d times", calls)
	}
}
