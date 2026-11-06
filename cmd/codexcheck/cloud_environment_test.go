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
