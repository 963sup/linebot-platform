// Command codexcheck validates repository-owned configuration without contacting MCP servers.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/963sup/linebot-platform/internal/governance/domainmodel"
	"github.com/BurntSushi/toml"
)

const maxAgentThreads = 9

var approvedAgentRoles = []string{"application", "architect", "domain", "environment", "integration", "platform", "researcher", "reviewer"}

var roleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type config struct {
	ProjectDocMaxBytes int `toml:"project_doc_max_bytes"`
	Features           struct {
		MultiAgent bool `toml:"multi_agent"`
	} `toml:"features"`
	Agents     map[string]any       `toml:"agents"`
	MCPServers map[string]mcpServer `toml:"mcp_servers"`
}

type roleRegistration struct{ Description, ConfigFile string }

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
	EnvHTTPHeaders           map[string]string `toml:"env_http_headers"`
	EnvVars                  []string          `toml:"env_vars"`
	EnabledTools             []string          `toml:"enabled_tools"`
	DefaultToolsApprovalMode string            `toml:"default_tools_approval_mode"`
}

type agent struct {
	Name                  string `toml:"name"`
	Description           string `toml:"description"`
	Model                 string `toml:"model"`
	DeveloperInstructions string `toml:"developer_instructions"`
}

func readTOML(path string, into any) error {
	md, err := toml.DecodeFile(path, into)
	if err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	keys := md.Undecoded()
	if c, ok := into.(*config); ok {
		// BurntSushi/toml leaves children of map[string]any marked undecoded.
		// Validate that dynamic table explicitly before allowing only its two
		// supported role fields; all other metadata remains subject to rejection.
		roles, err := parseAgentRegistry(c.Agents)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		unhandled := make([]toml.Key, 0, len(keys))
		for _, key := range keys {
			if len(key) == 3 && key[0] == "agents" {
				if _, exists := roles[key[1]]; exists && (key[2] == "description" || key[2] == "config_file") {
					continue
				}
			}
			unhandled = append(unhandled, key)
		}
		keys = unhandled
	}
	if len(keys) != 0 {
		return fmt.Errorf("%s has unsupported TOML keys: %v", path, keys)
	}
	return nil
}

func sameStrings(actual, wanted []string) bool {
	a, b := slices.Clone(actual), slices.Clone(wanted)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func parseAgentRegistry(values map[string]any) (map[string]roleRegistration, error) {
	enabled, ok := values["enabled"].(bool)
	n, nOK := values["max_concurrent_threads_per_session"].(int64)
	depth, depthOK := values["max_depth"].(int64)
	if !ok || !enabled || !nOK || n < 1 || n > maxAgentThreads || !depthOK || depth != 1 {
		return nil, fmt.Errorf("agents must be enabled with concurrency between 1 and %d", maxAgentThreads)
	}
	roles := make(map[string]roleRegistration)
	for name, value := range values {
		if name == "enabled" || name == "max_concurrent_threads_per_session" || name == "max_depth" {
			continue
		}
		table, ok := value.(map[string]any)
		if !ok || !roleNamePattern.MatchString(name) {
			return nil, fmt.Errorf("unsupported agents key %q", name)
		}
		for key := range table {
			if key != "description" && key != "config_file" {
				return nil, fmt.Errorf("agent %s: unsupported key %s", name, key)
			}
		}
		description, dOK := table["description"].(string)
		file, fOK := table["config_file"].(string)
		if !dOK || strings.TrimSpace(description) == "" || !fOK || file != "agents/"+name+".toml" {
			return nil, fmt.Errorf("agent %s requires a description and config_file agents/%s.toml", name, name)
		}
		roles[name] = roleRegistration{Description: description, ConfigFile: file}
	}
	if len(roles) == 0 {
		return nil, errors.New("agent registry is empty")
	}
	return roles, nil
}

var remoteEndpoints = map[string]string{
	"context7":                   "https://mcp.context7.com/mcp",
	"github":                     "https://api.githubcopilot.com/mcp/",
	"openaiDeveloperDocs":        "https://developers.openai.com/mcp",
	"google_developer_knowledge": "https://developerknowledge.googleapis.com/mcp",
	"google_cloud_cli":           "https://cloudcli.googleapis.com/mcp",
}

func validateConfig(path string) error {
	var c config
	if err := readTOML(path, &c); err != nil {
		return err
	}
	if !c.Features.MultiAgent || c.ProjectDocMaxBytes < 32768 {
		return errors.New("multi-agent and adequate project guidance capacity are required")
	}
	roles, err := parseAgentRegistry(c.Agents)
	if err != nil {
		return err
	}
	actualRoles := make([]string, 0, len(roles))
	for name := range roles {
		actualRoles = append(actualRoles, name)
	}
	if !sameStrings(actualRoles, approvedAgentRoles) {
		return fmt.Errorf("agent registry must contain exactly the approved delegated roles: %v", approvedAgentRoles)
	}
	expected := []string{"gopls", "repomix", "terraform", "line", "openapi_line", "firestore_toolbox"}
	for name := range remoteEndpoints {
		expected = append(expected, name)
	}
	actual := make([]string, 0, len(c.MCPServers))
	for name, server := range c.MCPServers {
		actual = append(actual, name)
		if server.Required || server.StartupTimeoutSec <= 0 || server.ToolTimeoutSec <= 0 {
			return fmt.Errorf("MCP %s must be optional with positive timeouts", name)
		}
		if len(server.HTTPHeaders) != 0 {
			return fmt.Errorf("MCP %s: literal HTTP headers are not allowed; use approved environment mappings", name)
		}
		if name != "google_developer_knowledge" && len(server.EnvHTTPHeaders) != 0 {
			return fmt.Errorf("MCP %s: unexpected environment header mapping", name)
		}
		if name != "github" && server.BearerTokenEnvVar != "" {
			return fmt.Errorf("MCP %s: unexpected bearer token mapping", name)
		}
		if name != "firestore_toolbox" && name != "google_cloud_cli" && name != "repomix" && name != "terraform" && name != "line" && name != "openapi_line" && (len(server.EnvVars) != 0 || len(server.EnabledTools) != 0) {
			return fmt.Errorf("MCP %s: unexpected tool or environment override", name)
		}
	}
	if !sameStrings(actual, expected) {
		return fmt.Errorf("MCP servers differ from the approved set: %v", actual)
	}
	gopls := c.MCPServers["gopls"]
	goplsArgs := []string{"-lc", "exec env XDG_CACHE_HOME=/workspace/.cache /workspace/.linebot-tools/bin/gopls mcp"}
	if !gopls.Enabled || gopls.Command != "bash" || !slices.Equal(gopls.Args, goplsArgs) || gopls.URL != "" || gopls.DefaultToolsApprovalMode != "" {
		return errors.New("gopls must use the pinned workspace path and writable XDG cache")
	}
	repomix := c.MCPServers["repomix"]
	repomixArgs := []string{"-lc", "exec env npm_config_cache=/workspace/.npm-cache npx -y repomix@1.18.1 --mcp --sandbox /workspace/linebot-platform"}
	repomixTools := []string{"pack_codebase", "read_repomix_output", "grep_repomix_output", "file_system_read_file", "file_system_read_directory"}
	if !repomix.Enabled || repomix.Command != "bash" || repomix.URL != "" || repomix.DefaultToolsApprovalMode != "" ||
		!slices.Equal(repomix.Args, repomixArgs) || !sameStrings(repomix.EnabledTools, repomixTools) ||
		len(repomix.EnvVars) != 0 || repomix.BearerTokenEnvVar != "" || len(repomix.EnvHTTPHeaders) != 0 {
		return errors.New("Repomix MCP must use the pinned workspace-confined read-only sandbox")
	}
	for name, endpoint := range remoteEndpoints {
		server := c.MCPServers[name]
		enabled := name == "context7" || name == "github" || name == "openaiDeveloperDocs" || name == "google_developer_knowledge" || name == "google_cloud_cli"
		if server.URL != endpoint || server.Command != "" || len(server.Args) != 0 || server.Enabled != enabled {
			return fmt.Errorf("MCP %s: unexpected endpoint, transport or enabled state", name)
		}
		approval := ""
		if !enabled {
			approval = "prompt"
		}
		if name == "github" {
			approval = "writes"
		}
		if name == "google_cloud_cli" {
			approval = "prompt"
		}
		if server.DefaultToolsApprovalMode != approval {
			return fmt.Errorf("MCP %s: unexpected approval policy", name)
		}
	}
	if c.MCPServers["github"].BearerTokenEnvVar != "GITHUB_MCP_TOKEN" {
		return errors.New("GitHub requires the GITHUB_MCP_TOKEN environment mapping")
	}
	knowledge := c.MCPServers["google_developer_knowledge"]
	if len(knowledge.EnvHTTPHeaders) != 1 || knowledge.EnvHTTPHeaders["X-Goog-Api-Key"] != "DEVELOPERKNOWLEDGE_API_KEY" {
		return errors.New("Google Developer Knowledge requires X-Goog-Api-Key mapped to DEVELOPERKNOWLEDGE_API_KEY, never a literal secret")
	}
	cloudCLI := c.MCPServers["google_cloud_cli"]
	if !sameStrings(cloudCLI.EnabledTools, []string{"run_gcloud_command"}) {
		return errors.New("Google Cloud CLI MCP must expose only its approval-gated command tool")
	}
	terra := c.MCPServers["terraform"]
	terraTools := []string{"search_provider_docs", "get_provider_details", "get_latest_provider_version", "get_provider_capabilities", "search_modules", "get_module_details", "get_latest_module_version", "search_policies", "get_policy_details"}
	if !terra.Enabled || terra.Command != "docker" || terra.URL != "" || terra.DefaultToolsApprovalMode != "prompt" ||
		!slices.Equal(terra.Args, []string{"run", "-i", "--rm", "hashicorp/terraform-mcp-server@sha256:11dd418bba6003dd5777e9d0979b68de59929a155b355f338f407947e00e9d80", "stdio", "--toolsets=registry"}) ||
		!sameStrings(terra.EnabledTools, terraTools) || len(terra.EnvVars) != 0 {
		return errors.New("Terraform MCP must use the pinned official image with public Registry tools only")
	}
	line := c.MCPServers["line"]
	lineTools := []string{"get_profile", "get_message_quota", "get_rich_menu_list", "get_follower_ids", "get_group_summary"}
	if !line.Enabled || line.Command != "npx" || line.URL != "" || line.DefaultToolsApprovalMode != "prompt" ||
		!slices.Equal(line.Args, []string{"-y", "@line/line-bot-mcp-server@0.5.0"}) ||
		!sameStrings(line.EnvVars, []string{"CHANNEL_ACCESS_TOKEN"}) ||
		!sameStrings(line.EnabledTools, lineTools) {
		return errors.New("LINE MCP must use the approved read-only tool set")
	}
	openAPI := c.MCPServers["openapi_line"]
	openAPIArgs := []string{"-lc", "exec env npm_config_cache=/workspace/.npm-cache npx -y @ivotoby/openapi-mcp-server@1.16.1 --api-base-url https://api.line.me --openapi-spec /workspace/.linebot-tools/openapi/line-messaging-api.yml --tools dynamic --verbose false"}
	if !openAPI.Enabled || openAPI.Command != "bash" || openAPI.URL != "" || openAPI.DefaultToolsApprovalMode != "prompt" ||
		!slices.Equal(openAPI.Args, openAPIArgs) || !sameStrings(openAPI.EnabledTools, []string{"list-api-endpoints", "get-api-endpoint-schema"}) || len(openAPI.EnvVars) != 0 {
		return errors.New("OpenAPI MCP must use the pinned LINE spec and omit endpoint invocation")
	}
	toolbox := c.MCPServers["firestore_toolbox"]
	if toolbox.Enabled || toolbox.Command != "npx" || toolbox.URL != "" || toolbox.DefaultToolsApprovalMode != "prompt" ||
		!slices.Equal(toolbox.Args, []string{"-y", "@toolbox-sdk/server@1.14.0", "--prebuilt=firestore", "--stdio"}) ||
		!sameStrings(toolbox.EnvVars, []string{"FIRESTORE_PROJECT", "GOOGLE_APPLICATION_CREDENTIALS"}) ||
		!sameStrings(toolbox.EnabledTools, []string{"get_documents", "query_collection", "list_collections", "get_rules", "validate_rules"}) {
		return errors.New("Firestore Toolbox must remain disabled with its pinned executable and restricted tools")
	}
	return nil
}

func validateRoleFiles(dir string, roles map[string]roleRegistration) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("agents directory must be a real directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("nested agent directories are unsupported: %s", entry.Name())
		}
		if filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("agent symlinks are forbidden: %s", entry.Name())
		}
		name := strings.TrimSuffix(entry.Name(), ".toml")
		if _, ok := roles[name]; !ok {
			return fmt.Errorf("unregistered agent file: %s", entry.Name())
		}
		var a agent
		if err := readTOML(filepath.Join(dir, entry.Name()), &a); err != nil {
			return err
		}
		if a.Name != name || strings.TrimSpace(a.Description) == "" || strings.TrimSpace(a.DeveloperInstructions) == "" {
			return fmt.Errorf("agent %s has invalid name, description or instructions", name)
		}
		if a.Model != strings.TrimSpace(a.Model) {
			return fmt.Errorf("agent %s has whitespace in model selection", name)
		}
		seen[name] = true
	}
	for name := range roles {
		if !seen[name] {
			return fmt.Errorf("registered agent file is missing: %s", name)
		}
	}
	return nil
}

func validateAgents(dir string) error {
	var c config
	if err := readTOML(filepath.Join(filepath.Dir(dir), "config.toml"), &c); err != nil {
		return err
	}
	roles, err := parseAgentRegistry(c.Agents)
	if err != nil {
		return err
	}
	return validateRoleFiles(dir, roles)
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
	if err := validateDomainModel(root); err != nil {
		return fmt.Errorf("domain model: %w", err)
	}
	for _, name := range []string{
		"scripts/check.sh",
		"scripts/codex-cloud/install.sh",
		"scripts/codex-cloud/preflight.sh",
		"scripts/codex-cloud/start.sh",
		"scripts/codex-cloud/verify.sh",
		"scripts/codex-cloud/repair.sh",
		"scripts/codex-cloud/lib/common.sh",
		"scripts/codex-cloud/lib/paths.sh",
		"scripts/codex-cloud/lib/skills.sh",
		"scripts/codex-cloud/lib/mcp.sh",
	} {
		cmd := exec.Command("bash", "-n", filepath.Join(root, name))
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("invalid %s shell syntax: %w: %s", name, err, strings.TrimSpace(string(output)))
		}
	}
	return rejectPythonFiles(root)
}

func validateDomainModel(root string) error {
	inventory, err := domainmodel.LoadEndpointInventory(filepath.Join(root, "docs", "line", "endpoint-inventory.yaml"))
	if err != nil {
		return err
	}
	model, err := domainmodel.LoadDomainModel(filepath.Join(root, "docs", "domain", "model.yaml"))
	if err != nil {
		return err
	}
	capabilities, err := domainmodel.LoadCapabilityMap(filepath.Join(root, "docs", "domain", "capability-map.yaml"))
	if err != nil {
		return err
	}
	dependencies, err := domainmodel.LoadDependencyCapabilities(filepath.Join(root, "docs", "domain", "dependency-capabilities.yaml"))
	if err != nil {
		return err
	}
	if err := domainmodel.ValidateRepository(root, inventory, model, capabilities, dependencies); err != nil {
		return err
	}
	want, err := domainmodel.RenderContextMap(model, capabilities, dependencies)
	if err != nil {
		return err
	}
	actual, err := os.ReadFile(filepath.Join(root, "docs", "domain", "context-map.md"))
	if err != nil {
		return err
	}
	if string(actual) != string(want) {
		return errors.New("docs/domain/context-map.md is stale; run go run ./cmd/domainmodel -mode=write")
	}
	return nil
}

func main() {
	root, err := os.Getwd()
	if err == nil {
		err = validate(root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Codex config validation:", err)
		os.Exit(1)
	}
	var c config
	if err := readTOML(filepath.Join(root, ".codex", "config.toml"), &c); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	roles, _ := parseAgentRegistry(c.Agents)
	fmt.Printf("Codex config OK: %d agents, %d MCP declarations; SDK catalog, retention, and domain governance verified. Runtime MCP/plugin availability is not verified.\n", len(roles), len(c.MCPServers))
}
