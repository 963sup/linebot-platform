package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPRuntimeSupplyChain(t *testing.T) {
	root := filepath.Join("..", "..")
	if err := validateMCPRuntimeSupplyChain(root); err != nil {
		t.Fatal(err)
	}
}

func TestMCPRuntimeRejectsManifestDrift(t *testing.T) {
	root := filepath.Join("..", "..")
	temp := copyMCPRuntimeInputs(t, root)
	path := filepath.Join(temp, "tools", "mcp-runtime", "package.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"repomix": "1.18.1"`, `"repomix": "latest"`, 1))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateMCPRuntimeSupplyChain(temp); err == nil {
		t.Fatal("unpinned MCP manifest accepted")
	}
}

func TestMCPRuntimeRejectsTransitiveLockDrift(t *testing.T) {
	root := filepath.Join("..", "..")
	temp := copyMCPRuntimeInputs(t, root)
	path := filepath.Join(temp, "tools", "mcp-runtime", "package-lock.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"resolved": "https://registry.npmjs.org/`, `"resolved": "https://packages.invalid/`, 1))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateMCPRuntimeSupplyChain(temp); err == nil {
		t.Fatal("transitive package with non-registry resolution accepted")
	}
}

func copyMCPRuntimeInputs(t *testing.T, root string) string {
	t.Helper()
	temp := t.TempDir()
	for _, name := range []string{"package.json", "package-lock.json"} {
		raw, err := os.ReadFile(filepath.Join(root, "tools", "mcp-runtime", name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(temp, "tools", "mcp-runtime", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range []string{"config/toolchain.lock", "scripts/codex-cloud/lib/mcp_runtime.sh"} {
		raw, err := os.ReadFile(filepath.Join(root, source))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(temp, source)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return temp
}
