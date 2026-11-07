package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffRuntimeContract(t *testing.T) {
	if err := validateHandoffRuntime(filepath.Join("..", "..")); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsUnpinnedPluginTask(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mise.toml"), []byte("codex plugin marketplace add google/skills\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := validateHandoffRuntime(root)
	if err == nil || !strings.Contains(err.Error(), "missing approved") {
		t.Fatalf("expected incomplete task contract rejection, got %v", err)
	}
}
