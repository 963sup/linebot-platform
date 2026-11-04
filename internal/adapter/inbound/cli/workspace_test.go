package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceOpsMustFailClosed(t *testing.T) {
	env := func(key string) string {
		if key == "APP_ENV" {
			return "local"
		}
		return ""
	}
	for _, args := range [][]string{
		{"--api", "calendar", "--read"},
		{"--api", "calendar", "--test-account", "--read", "--request", "nonexistent", "--token-file", "missing", "--out", "out"},
		{"--api", "tasks", "--test-account", "--request", "nonexistent", "--token-file", "missing", "--out", "out", "--apply", "--confirm", "tasks.delete"},
	} {
		if err := runWorkspace(args, env); err == nil {
			t.Fatalf("invalid workspace CLI accepted: %v", args)
		}
	}
	dir := t.TempDir()
	req := filepath.Join(dir, "request.json")
	token := filepath.Join(dir, "access_token.txt")
	if err := os.WriteFile(req, []byte(`{"operation":"events.insert","parameters":{"calendarId":"primary"},"body":{"summary":"Test"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(token, []byte("fake-test-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--api", "calendar", "--test-account", "--request", req, "--token-file", token, "--out", filepath.Join(dir, "output.json"), "--apply"}
	if err := runWorkspace(args, env); err == nil {
		t.Fatal("write without matching confirmation accepted")
	}
	if err := runWorkspace(append(args, "--confirm", "events.delete"), env); err == nil {
		t.Fatal("wrong write confirmation accepted")
	}
	prod := func(key string) string {
		if key == "APP_ENV" {
			return "production"
		}
		return ""
	}
	if err := runWorkspace(append(args, "--confirm", "events.insert"), prod); err == nil {
		t.Fatal("production environment accepted")
	}
	if err := os.Chmod(token, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := workspacePrivateFile(token, 1024); err == nil {
		t.Fatal("shared token file accepted")
	}
}
