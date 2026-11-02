package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexScenarioCatalog(t *testing.T) {
	if err := validateScenarios(filepath.Join("..", "..")); err != nil {
		t.Fatal(err)
	}
}

func TestCodexScenarioMissingManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.org/app\n\ngo 1.27.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateScenarios(root); err == nil {
		t.Fatal("missing acceptance catalog must fail closed")
	}
}
