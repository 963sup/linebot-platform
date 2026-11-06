package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var approvedMCPRuntimeDependencies = map[string]string{
	"@ivotoby/openapi-mcp-server": "1.16.1",
	"@line/line-bot-mcp-server":   "0.5.0",
	"@toolbox-sdk/server":         "1.14.0",
	"repomix":                     "1.18.1",
}

type npmManifest struct {
	Private      bool              `json:"private"`
	Dependencies map[string]string `json:"dependencies"`
}

type npmLock struct {
	LockfileVersion int `json:"lockfileVersion"`
	Packages        map[string]struct {
		Version      string            `json:"version"`
		Resolved     string            `json:"resolved"`
		Integrity    string            `json:"integrity"`
		Link         bool              `json:"link"`
		Dependencies map[string]string `json:"dependencies"`
	} `json:"packages"`
}

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
		MinVersion string            `toml:"min_version"`
		Env        map[string]string `toml:"env"`
		Tools      map[string]string `toml:"tools"`
		Tasks      map[string]struct {
			Run any `toml:"run"`
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
	dir := filepath.Join(root, "tools", "mcp-runtime")
	var manifest npmManifest
	if err := decodeJSONFile(filepath.Join(dir, "package.json"), &manifest); err != nil {
		return err
	}
	if !manifest.Private || !mapsEqual(manifest.Dependencies, approvedMCPRuntimeDependencies) {
		return fmt.Errorf("package.json must contain exactly the approved pinned MCP dependencies")
	}
	var lock npmLock
	if err := decodeJSONFile(filepath.Join(dir, "package-lock.json"), &lock); err != nil {
		return err
	}
	rootPackage, ok := lock.Packages[""]
	if lock.LockfileVersion != 3 || !ok || !mapsEqual(rootPackage.Dependencies, approvedMCPRuntimeDependencies) {
		return fmt.Errorf("package-lock root must pin the approved MCP dependencies with lockfileVersion 3")
	}
	for name, version := range approvedMCPRuntimeDependencies {
		pkg, ok := lock.Packages["node_modules/"+name]
		if !ok || pkg.Version != version || pkg.Resolved == "" || pkg.Integrity == "" {
			return fmt.Errorf("package-lock does not securely resolve %s@%s", name, version)
		}
	}
	for path, pkg := range lock.Packages {
		if path == "" || pkg.Link {
			continue
		}
		if !exactNPMVersion.MatchString(pkg.Version) || !strings.HasPrefix(pkg.Resolved, "https://registry.npmjs.org/") || pkg.Integrity == "" {
			return fmt.Errorf("package-lock entry %s must have exact version, npm registry HTTPS resolution and integrity", path)
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
	miseInstall, err := os.ReadFile(filepath.Join(root, "scripts", "codex-cloud", "lib", "mise.sh"))
	if err != nil {
		return err
	}
	for _, required := range []string{"https://github.com/jdx/mise/releases/download/v", "linux_x64_sha256", "sha256sum", "mv -Tf"} {
		if !strings.Contains(string(miseInstall), required) {
			return fmt.Errorf("mise installer is missing %q", required)
		}
	}
	install, err := os.ReadFile(filepath.Join(root, "scripts", "codex-cloud", "lib", "mcp_runtime.sh"))
	if err != nil {
		return err
	}
	text := string(install)
	for _, required := range []string{"npm ci --ignore-scripts", "mktemp -d", "receipt.json", "installed.sha256", "installed.symlinks", "sha256sum -c", `"$link_target" != /*`, "readlink -f", "lock_sha256", "find \"$temp\" -type f -exec chmod a-w", "mv -T -- \"$temp\" \"$release\"", "codex_cloud_mcp_release_check \"$release\"", "mv -Tf -- \"$pointer\" \"$target\""} {
		if !strings.Contains(text, required) {
			return fmt.Errorf("MCP runtime installer is missing %q", required)
		}
	}
	if strings.Contains(text, "trap ") && strings.Contains(text, "RETURN") {
		return fmt.Errorf("MCP runtime installer must not use a RETURN trap for temporary cleanup")
	}
	return nil
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range b {
		if a[key] != value {
			return false
		}
	}
	return true
}
