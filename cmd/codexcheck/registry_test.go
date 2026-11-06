package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentRegistryBoundaries(t *testing.T) {
	for _, n := range []int64{-1, 0, 1, 4, 8, 9, 10} {
		_, err := parseAgentRegistry(map[string]any{
			"enabled": true, "max_concurrent_threads_per_session": n, "max_depth": int64(1),
			"example": map[string]any{"description": "test", "config_file": "agents/example.toml"},
		})
		if (err == nil) != (n >= 1 && n <= 9) {
			t.Errorf("threads %d: %v", n, err)
		}
	}
	for _, depth := range []int64{0, 1, 2} {
		_, err := parseAgentRegistry(map[string]any{
			"enabled": true, "max_concurrent_threads_per_session": int64(9), "max_depth": depth,
			"example": map[string]any{"description": "test", "config_file": "agents/example.toml"},
		})
		if (err == nil) != (depth == 1) {
			t.Errorf("depth %d: %v", depth, err)
		}
	}
	for _, bad := range []map[string]any{
		{"description": "test", "config_file": "../outside.toml"},
		{"description": "test", "config_file": "agents/other.toml"},
		{"description": "test", "config_file": "agents/example.toml", "unknown": true},
		{"description": "", "config_file": "agents/example.toml"},
	} {
		_, err := parseAgentRegistry(map[string]any{"enabled": true, "max_concurrent_threads_per_session": int64(9), "max_depth": int64(1), "example": bad})
		if err == nil {
			t.Errorf("accepted invalid role: %v", bad)
		}
	}
}

func TestAgentModelAndUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.toml")
	valid := "name = \"example\"\ndescription = \"test\"\nmodel = \"existing-host-model\"\ndeveloper_instructions = \"test\"\n"
	if err := os.WriteFile(path, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	var a agent
	if err := readTOML(path, &a); err != nil {
		t.Fatal(err)
	}
	if a.Model != "existing-host-model" {
		t.Fatal("model lost")
	}
	if err := os.WriteFile(path, []byte(valid+"unknown_key = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := readTOML(path, &a); err == nil {
		t.Fatal("unknown key accepted")
	}
}

func TestRegistryMatchesFilesAndRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "agents")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	reg := map[string]roleRegistration{"example": {Description: "test", ConfigFile: "agents/example.toml"}}
	if err := validateRoleFiles(dir, reg); err == nil {
		t.Fatal("missing role accepted")
	}
	body := "name = \"example\"\ndescription = \"test\"\ndeveloper_instructions = \"Use Repomix MCP includePatterns with a baseline and same-scope post-change pack.\"\n"
	path := filepath.Join(dir, "example.toml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateRoleFiles(dir, reg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "orphan.toml"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateRoleFiles(dir, reg); err == nil {
		t.Fatal("orphan accepted")
	}
	if err := os.Remove(filepath.Join(dir, "orphan.toml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(root, "external.toml")
	if err := os.WriteFile(external, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := validateRoleFiles(dir, reg); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestRetainApprovedRoles(t *testing.T) {
	var c config
	if err := readTOML(filepath.Join("..", "..", ".codex", "config.toml"), &c); err != nil {
		t.Fatal(err)
	}
	roles, err := parseAgentRegistry(c.Agents)
	if err != nil {
		t.Fatal(err)
	}
	// This regression assertion protects the V3 role set; production parsing stays data-driven.
	for _, name := range strings.Fields(`architect architecture_auditor researcher domain application integration platform environment reviewer`) {
		if _, ok := roles[name]; !ok {
			t.Errorf("approved role %s missing", name)
		}
	}
	if len(roles) != 9 {
		t.Fatalf("expected exactly nine delegated roles, got %d", len(roles))
	}
	if _, ok := roles["orchestrator"]; ok {
		t.Fatal("primary orchestrator must not be registered as a delegated role")
	}
}

func TestRejectConfigDrift(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, from, to string }{
		{"threads", "max_concurrent_threads_per_session = 9", "max_concurrent_threads_per_session = 10"},
		{"knowledge-secret", `env_http_headers = { "X-Goog-Api-Key" = "DEVELOPERKNOWLEDGE_API_KEY" }`, `http_headers = { "X-Goog-Api-Key" = "do-not-store-keys" }`},
		{"repomix-runtime", `"repomix", "--mcp"`, `"npx", "--mcp"`},
		{"line-runtime", "npm:@line/line-bot-mcp-server@0.5.0", "npx"},
		{"terraform-digest", "hashicorp/terraform-mcp-server@sha256:11dd418bba6003dd5777e9d0979b68de59929a155b355f338f407947e00e9d80", "hashicorp/terraform-mcp-server:1.4.1"},
		{"repomix-sandbox", `"--sandbox", "/workspace/linebot-platform"`, `"--sandbox", "/"`},
		{"bootstrap-path", "/workspace/linebot-platform/bootstrap.sh", "/tmp/bootstrap.sh"},
		{"unknown", "project_doc_max_bytes = 65536", "project_doc_max_bytes = 65536\nunknown_key = true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(string(data), tc.from) {
				t.Fatalf("test precondition missing: %s", tc.from)
			}
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(strings.Replace(string(data), tc.from, tc.to, 1)), 0600); err != nil {
				t.Fatal(err)
			}
			if err := validateConfig(path); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestValidateConfigRejectsDelegatedOrchestrator(t *testing.T) {
	data := mustRead(filepath.Join("..", "..", ".codex", "config.toml"))
	addition := "\n[agents.orchestrator]\ndescription = \"must remain primary\"\nconfig_file = \"agents/orchestrator.toml\"\n"
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, append(data, []byte(addition)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateConfig(path); err == nil || !strings.Contains(err.Error(), "exactly the approved") {
		t.Fatalf("delegated orchestrator error = %v", err)
	}
}
