package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var exactNPMVersion = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

func validateMCPRuntimeSupplyChain(root string) error {
	var miseConfig struct {
		Settings struct {
			ExecAutoInstall *bool `toml:"exec_auto_install"`
		} `toml:"settings"`
		MinVersion string            `toml:"min_version"`
		Env        map[string]string `toml:"env"`
		Tools      map[string]string `toml:"tools"`
		Tasks      map[string]struct {
			Description string            `toml:"description"`
			Run         any               `toml:"run"`
			Env         map[string]string `toml:"env"`
		} `toml:"tasks"`
	}
	if err := readTOML(filepath.Join(root, "mise.toml"), &miseConfig); err != nil {
		return err
	}
	for name, version := range miseConfig.Tools {
		if !exactNPMVersion.MatchString(version) {
			return fmt.Errorf("mise tool %s must have an exact version", name)
		}
	}
	for _, name := range []string{"go", "node", "go:golang.org/x/tools/gopls", "npm:repomix", "npm:@modelcontextprotocol/server-filesystem"} {
		if miseConfig.Tools[name] == "" {
			return fmt.Errorf("mise is missing required tool %s", name)
		}
	}
	if miseConfig.Tools["npm:@modelcontextprotocol/server-filesystem"] != "2.2.0" {
		return fmt.Errorf("Filesystem MCP install version must match the configured 2.2.0 runtime")
	}
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	goVersion := regexp.MustCompile(`(?m)^go ([0-9.]+)$`).FindStringSubmatch(string(goMod))
	if len(goVersion) != 2 || miseConfig.Tools["go"] != goVersion[1] {
		return fmt.Errorf("mise Go version must match go.mod")
	}
	if !exactNPMVersion.MatchString(miseConfig.MinVersion) {
		return fmt.Errorf("mise min_version must be exact")
	}
	if miseConfig.Settings.ExecAutoInstall == nil || *miseConfig.Settings.ExecAutoInstall {
		return fmt.Errorf("MCP execution must not auto-install tools")
	}
	bootstrap, err := os.ReadFile(filepath.Join(root, "bootstrap.sh"))
	if err != nil {
		return err
	}
	text := string(bootstrap)
	for _, forbidden := range []string{"git reset", "git clean", "CODEX_HOME", "$HOME/.codex"} {
		if strings.Contains(text, forbidden) {
			return fmt.Errorf("bootstrap must not contain %q", forbidden)
		}
	}
	if !regexp.MustCompile(`(?m)printf .* [0-9a-f]{64} .*sha256sum -c`).MatchString(text) {
		return fmt.Errorf("bootstrap must verify the mise download with a pinned SHA-256")
	}
	for _, required := range []string{"https://github.com/jdx/mise/releases/download/v" + miseConfig.MinVersion, "sha256sum -c", "MISE_EXEC_AUTO_INSTALL=0"} {
		if !strings.Contains(text, required) {
			return fmt.Errorf("bootstrap missing verified mise setup contract %q", required)
		}
	}
	return nil
}
