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
		MinVersion string              `toml:"min_version"`
		Env        map[string]string   `toml:"env"`
		Tools      map[string]miseTool `toml:"tools"`
		Tasks      map[string]struct {
			Description string            `toml:"description"`
			Run         any               `toml:"run"`
			Env         map[string]string `toml:"env"`
		} `toml:"tasks"`
	}
	if err := readTOML(filepath.Join(root, "mise.toml"), &miseConfig); err != nil {
		return err
	}
	for name, tool := range miseConfig.Tools {
		if !exactNPMVersion.MatchString(tool.Version) {
			return fmt.Errorf("mise tool %s must have an exact version", name)
		}
		if (name == "pipx:graphifyy" && tool.Extras != "mcp") || (name != "pipx:graphifyy" && tool.Extras != "") {
			return fmt.Errorf("only Graphify may specify extras, and it requires mcp")
		}
	}
	for _, name := range []string{"go", "node", "python", "uv", "pipx:graphifyy", "go:golang.org/x/tools/gopls", "npm:repomix", "npm:@modelcontextprotocol/server-filesystem", "npm:chrome-devtools-mcp", "npm:@playwright/mcp"} {
		if miseConfig.Tools[name].Version == "" {
			return fmt.Errorf("mise is missing required tool %s", name)
		}
	}
	if miseConfig.Tools["npm:@modelcontextprotocol/server-filesystem"].Version != "2.2.0" {
		return fmt.Errorf("Filesystem MCP install version must match the configured 2.2.0 runtime")
	}
	if miseConfig.Env["GRAPHIFY_BACKEND"] != "none" || miseConfig.Env["GRAPHIFY_QUERY_LOG_DISABLE"] != "1" {
		return fmt.Errorf("Graphify must disable LLM backends and query logging")
	}
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	goVersion := regexp.MustCompile(`(?m)^go ([0-9.]+)$`).FindStringSubmatch(string(goMod))
	if len(goVersion) != 2 || miseConfig.Tools["go"].Version != goVersion[1] {
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

// mise supports a version string or a table for tools requiring install extras.
type miseTool struct {
	Version string
	Extras  string
}

func (t *miseTool) UnmarshalTOML(value any) error {
	if version, ok := value.(string); ok {
		t.Version = version
		return nil
	}
	options, ok := value.(map[string]any)
	if !ok || len(options) != 2 {
		return fmt.Errorf("unsupported mise tool options")
	}
	version, versionOK := options["version"].(string)
	extras, extrasOK := options["extras"].(string)
	if !versionOK || !extrasOK {
		return fmt.Errorf("mise tool requires string version and extras")
	}
	t.Version, t.Extras = version, extras
	return nil
}
