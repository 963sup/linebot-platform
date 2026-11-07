package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloudInstallLeavesHostCodexConfigUntouched(t *testing.T) {
	root := filepath.Join("..", "..")
	paths := []string{"bootstrap.sh"}
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

}

func TestBootstrapOwnsDerivedWorkspacePaths(t *testing.T) {
	root := filepath.Join("..", "..")
	bootstrap, err := os.ReadFile(filepath.Join(root, "bootstrap.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(bootstrap)
	for _, marker := range []string{"BOOTSTRAP_WORKSPACE", "BOOTSTRAP_REPO", "XDG_CACHE_HOME", "NPM_CONFIG_CACHE", "GOCACHE", "GOPATH", "CLOUDSDK_CONFIG"} {
		if !strings.Contains(text, marker) {
			t.Errorf("bootstrap missing derived path %q", marker)
		}
	}
	for _, unnecessary := range []string{"BOOTSTRAP_AGENT_HOME", "LINEBOT_TOOL_ROOT", "LINEBOT_CACHE_ROOT", "LINEBOT_CONFIG_ROOT", "LINEBOT_STATE_ROOT"} {
		if strings.Contains(text, unnecessary) {
			t.Errorf("bootstrap retains unnecessary derived alias %q", unnecessary)
		}
	}
	mise, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, duplicate := range []string{"/workspace/.cache", "/workspace/.config", "/workspace/.local", "/workspace/.tools"} {
		if strings.Contains(string(mise), duplicate) {
			t.Errorf("mise.toml duplicates bootstrap-owned path %q", duplicate)
		}
	}
}
