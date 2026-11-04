package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestSDKRetentionDoesNotRequireEmptyScaffoldDirectories(t *testing.T) {
	root := t.TempDir()
	writeProtectedGoMod(t, root)
	manifest := filepath.Join(root, "internal", "infrastructure", "dependencies", "modules.go")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte("//go:build dependencies\n\npackage dependencies\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateSDKRetention(root); err != nil {
		t.Fatalf("empty SDK scaffold directories must not be required: %v", err)
	}
}

func TestSDKRetentionRequiresBuildTagManifest(t *testing.T) {
	root := t.TempDir()
	writeProtectedGoMod(t, root)
	if err := validateSDKRetention(root); err == nil {
		t.Fatal("removing the build-tag dependency manifest must fail validation")
	}
}

func writeProtectedGoMod(t *testing.T, root string) {
	t.Helper()
	var goMod strings.Builder
	goMod.WriteString("module example.org/example\n\ngo 1.27.3\n\nrequire (\n")
	for _, module := range protectedSDKModules {
		goMod.WriteString("\t")
		goMod.WriteString(module)
		goMod.WriteString(" v0.0.0\n")
	}
	goMod.WriteString(")\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}
