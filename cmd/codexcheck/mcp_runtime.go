package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var exactNPMVersion = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

func decodeJSONFile(path string, into any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

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
	var toolchain struct {
		Go   string `json:"go"`
		Mise struct {
			Version        string `json:"version"`
			LinuxX64SHA256 string `json:"linux_x64_sha256"`
		} `json:"mise"`
		Gopls struct {
			Version string `json:"version"`
		} `json:"gopls"`
	}
	if err := decodeJSONFile(filepath.Join(root, "config", "toolchain.lock"), &toolchain); err != nil {
		return err
	}
	if toolchain.Go != "go1.27.3" || toolchain.Mise.Version != "2026.11.3" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(toolchain.Mise.LinuxX64SHA256) {
		return fmt.Errorf("Go and mise must use the approved pinned toolchain")
	}
	if toolchain.Gopls.Version != "v0.24.0" {
		return fmt.Errorf("gopls must be pinned to v0.24.0")
	}
	if miseConfig.MinVersion != toolchain.Mise.Version || miseConfig.Tools["go"] != strings.TrimPrefix(toolchain.Go, "go") || miseConfig.Tools["go:golang.org/x/tools/gopls"] != strings.TrimPrefix(toolchain.Gopls.Version, "v") || miseConfig.Tools["npm:repomix"] != "1.18.1" {
		return fmt.Errorf("mise must own the approved Go, gopls and Repomix versions")
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
	for _, required := range []string{"https://github.com/jdx/mise/releases/download/v" + toolchain.Mise.Version, toolchain.Mise.LinuxX64SHA256, "sha256sum -c", "MISE_EXEC_AUTO_INSTALL=0"} {
		if !strings.Contains(text, required) {
			return fmt.Errorf("bootstrap missing verified mise setup contract %q", required)
		}
	}
	return nil
}
