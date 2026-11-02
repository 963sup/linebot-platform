package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectModules(t *testing.T) {
	file := filepath.Join(t.TempDir(), "go.mod")
	content := `module example.org/app

go 1.27.3
require (
	example.org/a v1.2.3
	example.org/indirect v4.5.6 // indirect
)
require example.org/b v2.0.0
`
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := directModules(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["example.org/a"] != "v1.2.3" || got["example.org/b"] != "v2.0.0" {
		t.Fatalf("direct modules: %#v", got)
	}
}

func writeCatalogFixture(t *testing.T, root, module, imp string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "docs", "sdk"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.org/app\n\ngo 1.27.3\nrequire example.org/sdk v1.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog := `schema_version = 1

[[modules]]
module = "` + module + `"
provider = "example"
kind = "utility"
status = "integrated"
imports = ["` + imp + `"]
capabilities = ["example"]
reference = "https://pkg.go.dev/example.org/sdk"
`
	if err := os.WriteFile(filepath.Join(root, "docs", "sdk", "catalog.toml"), []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogValidatesDirectModules(t *testing.T) {
	root := t.TempDir()
	writeCatalogFixture(t, root, "example.org/sdk", "example.org/sdk")
	if err := validateCatalog(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.org/app\n\ngo 1.27.3\nrequire (\n example.org/sdk v1.0.0\n example.org/new v1.2.3\n)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateCatalog(root); err == nil || !strings.Contains(err.Error(), "missing direct go.mod") {
		t.Fatalf("expected a missing-module error, got %v", err)
	}
}

func TestCatalogRejectsInvalidImports(t *testing.T) {
	root := t.TempDir()
	writeCatalogFixture(t, root, "example.org/sdk", "example.org/other")
	if err := validateCatalog(root); err == nil || !strings.Contains(err.Error(), "outside module") {
		t.Fatalf("expected invalid import error, got %v", err)
	}
}

func TestCatalogRejectsUndeclaredModule(t *testing.T) {
	root := t.TempDir()
	writeCatalogFixture(t, root, "example.org/unknown", "example.org/unknown")
	if err := validateCatalog(root); err == nil || !strings.Contains(err.Error(), "not a direct") {
		t.Fatalf("expected undeclared module error, got %v", err)
	}
}

func TestIntegratedServiceRequiresImplementation(t *testing.T) {
	root := t.TempDir()
	writeCatalogFixture(t, root, "example.org/sdk", "example.org/sdk")
	catalogPath := filepath.Join(root, "docs", "sdk", "catalog.toml")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(data), "kind = \"utility\"",
		"kind = \"service\"\nfactory = \"internal/infrastructure/example\"\nadapter = \"internal/adapter/outbound/example\"", 1)
	if err := os.WriteFile(catalogPath, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateCatalog(root); err == nil || !strings.Contains(err.Error(), "no Go implementation") {
		t.Fatalf("expected missing implementation error, got %v", err)
	}
}
