package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryConfiguration(t *testing.T) {
	if err := validate(filepath.Join("..", "..")); err != nil {
		t.Fatal(err)
	}
}

func TestRejectMalformedTOML(t *testing.T) {
	file := filepath.Join(t.TempDir(), "invalid.toml")
	if err := os.WriteFile(file, []byte("[mcp_servers.gopls\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateConfig(file); err == nil {
		t.Fatal("expected invalid TOML to fail")
	}
}

func TestRejectPythonFiles(t *testing.T) {
	root := t.TempDir()
	if err := rejectPythonFiles(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "unexpected.py")
	if err := os.WriteFile(path, []byte("pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := rejectPythonFiles(root); err == nil || !strings.Contains(err.Error(), "unexpected.py") {
		t.Fatalf("expected Python source rejection, got %v", err)
	}
}
