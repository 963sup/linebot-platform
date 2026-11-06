package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloudInstallLeavesHostCodexConfigUntouched(t *testing.T) {
	root := filepath.Join("..", "..")
	paths := []string{
		"scripts/codex-cloud/install.sh",
		"scripts/codex-cloud/verify.sh",
		"bootstrap.sh",
		"scripts/codex-cloud/start.sh",
	}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, forbidden := range []string{"CODEX_HOME", "$HOME/.codex", "trust_level", "git reset", "git clean"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s must not manage Host-owned Codex configuration: found %q", path, forbidden)
			}
		}
	}

	install, err := os.ReadFile(filepath.Join(root, "scripts/codex-cloud/install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(install), "codex_cloud_mcp_registration_check") {
		t.Error("install must defer Host MCP registration checks to a fresh task")
	}

	verify, err := os.ReadFile(filepath.Join(root, "scripts/codex-cloud/verify.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(verify), "mcp-check") {
		t.Error("verify must run the MCP protocol checks")
	}
}
