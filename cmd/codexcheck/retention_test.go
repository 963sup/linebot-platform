package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReservedSDKInventoryMatchesRepository(t *testing.T) {
	if err := validateSDKRetention(filepath.Join("..", "..")); err != nil {
		t.Fatal(err)
	}
}

func TestReservedSDKModulesCannotBeSilentlyPruned(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.org/example\n\ngo 1.27.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateSDKRetention(root); err == nil {
		t.Fatal("removing protected direct SDK modules must fail validation")
	}
}
