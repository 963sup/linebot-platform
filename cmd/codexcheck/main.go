// Command codexcheck validates repository-owned Codex configuration.
// It runs without Python and never connects to external MCP servers.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

type config struct {
	Features struct {
		MultiAgent bool `toml:"multi_agent"`
	} `toml:"features"`
	Agents struct {
		Enabled                        bool `toml:"enabled"`
		MaxConcurrentThreadsPerSession int  `toml:"max_concurrent_threads_per_session"`
	} `toml:"agents"`
	MCPServers map[string]mcpServer `toml:"mcp_servers"`
}

type mcpServer struct {
	Command                  string            `toml:"command"`
	Args                     []string          `toml:"args"`
	URL                      string            `toml:"url"`
	Enabled                  bool              `toml:"enabled"`
	Required                 bool              `toml:"required"`
	StartupTimeoutSec        int               `toml:"startup_timeout_sec"`
	ToolTimeoutSec           int               `toml:"tool_timeout_sec"`
	BearerTokenEnvVar        string            `toml:"bearer_token_env_var"`
	HTTPHeaders              map[string]string `toml:"http_headers"`
	EnvVars                  []string          `toml:"env_vars"`
	EnabledTools             []string          `toml:"enabled_tools"`
	DefaultToolsApprovalMode string            `toml:"default_tools_approval_mode"`
}

type agent struct {
	Name                  string `toml:"name"`
	Description           string `toml:"description"`
	DeveloperInstructions string `toml:"developer_instructions"`
}

func readTOML(path string, into any) error {
	md, err := toml.DecodeFile(path, into)
	if err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) != 0 {
		return fmt.Errorf("%s has unsupported TOML keys: %v", path, undecoded)
	}
	return nil
}

func sameStrings(actual, wanted []string) bool {
	if len(actual) != len(wanted) {
		return false
	}
	a := slices.Clone(actual)
	b := slices.Clone(wanted)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func validateConfig(path string) error {
	var c config
	if err := readTOML(path, &c); err != nil {
		return err
	}
	if !c.Features.MultiAgent || !c.Agents.Enabled ||
		c.Agents.MaxConcurrentThreadsPerSession < 1 ||
		c.Agents.MaxConcurrentThreadsPerSession > 3 {
		return errors.New("multi-agent must be enabled with concurrency between 1 and 3")
	}
	expected := []string{"gopls", "context7", "github_readonly", "firestore_toolbox", "google_cloud_run", "google_firestore", "openaiDeveloperDocs", "google_developer_knowledge", "google_cloud_cli", "google_cloud_logging", "google_cloud_monitoring", "google_calendar", "google_drive", "google_sheets", "google_docs", "google_gmail"}
	keys := make([]string, 0, len(c.MCPServers))
	for name, s := range c.MCPServers {
		keys = append(keys, name)
		if s.Required || s.StartupTimeoutSec <= 0 || s.ToolTimeoutSec <= 0 {
			return fmt.Errorf("MCP %s must be optional with positive timeouts", name)
		}
	}
	if !sameStrings(keys, expected) {
		return fmt.Errorf("MCP servers differ from the allowed set: %v", keys)
	}
	gopls := c.MCPServers["gopls"]
	if !gopls.Enabled || gopls.Command != "bash" ||
		len(gopls.Args) != 2 || gopls.Args[0] != "-c" ||
		!strings.Contains(gopls.Args[1], "/.local/bin/gopls") ||
		!strings.HasSuffix(gopls.Args[1], " mcp") {
		return errors.New("gopls MCP must use the Cloud-installed local gopls mcp executable")
	}
	context7 := c.MCPServers["context7"]
	if !context7.Enabled || context7.URL != "https://mcp.context7.com/mcp" {
		return errors.New("Context7 MCP must use the documented HTTPS endpoint")
	}
	docs := c.MCPServers["openaiDeveloperDocs"]
	if !docs.Enabled || docs.URL != "https://developers.openai.com/mcp" ||
		docs.BearerTokenEnvVar != "" || docs.Command != "" {
		return errors.New("OpenAI Docs MCP must use the official HTTPS endpoint without local command or bearer token")
	}
	github := c.MCPServers["github_readonly"]
	if !github.Enabled || github.URL != "https://api.githubcopilot.com/mcp/" ||
		github.BearerTokenEnvVar != "GITHUB_MCP_TOKEN" ||
		github.HTTPHeaders["X-MCP-Readonly"] != "true" {
		return errors.New("GitHub MCP must be enabled and explicitly read-only")
	}
	firestore := c.MCPServers["firestore_toolbox"]
	if firestore.Enabled || firestore.Command != "npx" ||
		len(firestore.Args) < 2 ||
		!sameStrings(firestore.Args[len(firestore.Args)-2:], []string{"--prebuilt=firestore", "--stdio"}) ||
		!sameStrings(firestore.EnabledTools, []string{
			"get_documents", "query_collection", "list_collections", "get_rules", "validate_rules",
		}) {
		return errors.New("Firestore Toolbox must remain disabled with a restricted tool list")
	}
	for name, endpoint := range map[string]string{
		"google_cloud_run": "https://run.googleapis.com/mcp",
		"google_firestore": "https://firestore.googleapis.com/mcp",
	} {
		server := c.MCPServers[name]
		if server.Enabled || server.URL != endpoint || server.Command != "" ||
			server.BearerTokenEnvVar != "" || len(server.HTTPHeaders) != 0 {
			return fmt.Errorf("%s must remain a disabled, credential-free Google remote MCP", name)
		}
	}
	for name, endpoint := range map[string]string{
		"google_cloud_cli":        "https://cloudcli.googleapis.com/mcp",
		"google_cloud_logging":    "https://logging.googleapis.com/mcp",
		"google_cloud_monitoring": "https://monitoring.googleapis.com/mcp",
		"google_calendar":         "https://calendarmcp.googleapis.com/mcp/v1",
		"google_drive":            "https://drivemcp.googleapis.com/mcp/v1",
		"google_sheets":           "https://sheetsmcp.googleapis.com/mcp/v1",
		"google_docs":             "https://docsmcp.googleapis.com/mcp/v1",
		"google_gmail":            "https://gmailmcp.googleapis.com/mcp/v1",
	} {
		server := c.MCPServers[name]
		if server.Enabled || server.URL != endpoint || server.Command != "" ||
			server.BearerTokenEnvVar != "" || len(server.HTTPHeaders) != 0 {
			return fmt.Errorf("%s must remain a disabled, credential-free Google remote MCP", name)
		}
	}
	knowledge := c.MCPServers["google_developer_knowledge"]
	if !knowledge.Enabled || knowledge.URL != "https://developerknowledge.googleapis.com/mcp" ||
		knowledge.Command != "" || knowledge.BearerTokenEnvVar != "" || len(knowledge.HTTPHeaders) != 0 {
		return errors.New("Google Developer Knowledge MCP must use its configured HTTPS endpoint without embedded credentials")
	}
	return nil
}

func validateAgents(dir string) error {
	expected := []string{
		"domain_architect", "line_integrator", "cloud_platform", "go_reviewer", "docs_researcher",
		"code_cleanup_release_validation", "gopls_go_refactor", "google_identity_workspace_integrator",
		"delivery_orchestrator", "go_feature_engineer", "go_architecture_reviewer", "github_actions_reviewer",
		"go_agents_md_author", "technical_docs_writer", "cloud_run_release",
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.toml"))
	if err != nil {
		return fmt.Errorf("list agent files: %w", err)
	}
	names := make([]string, 0, len(files))
	for _, file := range files {
		var a agent
		if err := readTOML(file, &a); err != nil {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(file), ".toml")
		if a.Name != name || strings.TrimSpace(a.Description) == "" ||
			strings.TrimSpace(a.DeveloperInstructions) == "" {
			return fmt.Errorf("%s is missing a valid name, description, or developer instructions", file)
		}
		names = append(names, name)
	}
	if !sameStrings(names, expected) {
		return fmt.Errorf("unexpected Codex agent files: %v", names)
	}
	return nil
}

func rejectPythonFiles(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", ".venv", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".py") {
			return fmt.Errorf("Python source is not permitted in this repository: %s", path)
		}
		return nil
	})
}

func validate(root string) error {
	if err := validateConfig(filepath.Join(root, ".codex", "config.toml")); err != nil {
		return err
	}
	if err := validateAgents(filepath.Join(root, ".codex", "agents")); err != nil {
		return err
	}
	if err := validateCatalog(root); err != nil {
		return fmt.Errorf("SDK catalog: %w", err)
	}
	if err := validateSDKRetention(root); err != nil {
		return fmt.Errorf("SDK retention: %w", err)
	}
	setup := filepath.Join(root, "scripts", "setup-codex-cloud.sh")
	cmd := exec.Command("bash", "-n", setup)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("invalid setup shell syntax: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return rejectPythonFiles(root)
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Codex config validation:", err)
		os.Exit(1)
	}
	if err := validate(root); err != nil {
		fmt.Fprintln(os.Stderr, "Codex config validation:", err)
		os.Exit(1)
	}
	fmt.Println("Codex config OK: 15 agents, 16 MCP entries, SDK catalog and reserved SDKs; no Python source files")
}
