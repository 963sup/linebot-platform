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

func copyMCPRuntimeInputs(t *testing.T, root string) string {
	t.Helper()
	temp := t.TempDir()
	for _, source := range []string{"config/toolchain.lock", "mise.toml", "bootstrap.sh"} {
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

func TestMiseRejectsRuntimeInstallAndVersionDrift(t *testing.T) {
	for _, tc := range []struct{ name, from, to string }{
		{"runtime-install", "exec_auto_install = false", "exec_auto_install = true"},
		{"repomix-version", `"npm:repomix" = "1.18.1"`, `"npm:repomix" = "latest"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := copyMCPRuntimeInputs(t, filepath.Join("..", ".."))
			if err := validateMCPRuntimeSupplyChain(root); err != nil {
				t.Fatalf("invalid baseline: %v", err)
			}
			path := filepath.Join(root, "mise.toml")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), tc.from) {
				t.Fatal("missing mutation target")
			}
			if err := os.WriteFile(path, []byte(strings.Replace(string(raw), tc.from, tc.to, 1)), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := validateMCPRuntimeSupplyChain(root); err == nil {
				t.Fatal("unsafe mise configuration accepted")
			}
		})
	}
}
