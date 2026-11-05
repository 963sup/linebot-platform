package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuidanceTracksRegisteredCapabilities(t *testing.T) {
	root := filepath.Join("..", "..")
	var c config
	if err := readTOML(filepath.Join(root, ".codex", "config.toml"), &c); err != nil {
		t.Fatal(err)
	}
	roles, err := parseAgentRegistry(c.Agents)
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	roleDoc := read("docs/ai/agents.md")
	for name := range roles {
		if !strings.Contains(roleDoc, "`"+name+"`") {
			t.Errorf("role %s missing from matrix", name)
		}
	}
	mcpDoc := read("docs/ai/mcp-and-plugins.md")
	for name, server := range c.MCPServers {
		state := "configured-disabled"
		if server.Enabled {
			state = "configured-enabled"
		}
		if !strings.Contains(mcpDoc, "| `"+name+"` | `"+state+"` |") {
			t.Errorf("MCP %s state missing or stale", name)
		}
	}
	for _, path := range []string{"AGENTS.md", "docs/ai/agents.md", ".codex/agents/architect.toml"} {
		if strings.Contains(read(path), "DIRECTORY-STRUCTURE.md") {
			t.Errorf("%s references deleted architecture target", path)
		}
	}
	for _, path := range []string{"AGENTS.md", "docs/agent-workflows/routing-policy.md", "docs/ai/agents.md"} {
		guidance := read(path)
		if !strings.Contains(guidance, "nine") || !strings.Contains(guidance, "depth is one") {
			t.Errorf("%s must state the nine-thread and depth-one limits", path)
		}
	}
	acceptance := read("docs/ai/agents-mcp-plugins-acceptance.md")
	for _, want := range []string{"8 個角色", "11 宣告、10 啟用、1 個服務維持停用", "1／4／8／9", "0／10"} {
		if !strings.Contains(acceptance, want) {
			t.Errorf("acceptance guidance missing current baseline %q", want)
		}
	}
	for _, path := range []string{"AGENTS.md", "docs/ai/agents.md", ".agents/skills/agent-tool-preflight/SKILL.md"} {
		guidance := read(path)
		for _, want := range []string{"includePatterns", "**/*.md", "**/*.{go,mod,sum}", "Not Verified"} {
			if !strings.Contains(guidance, want) {
				t.Errorf("%s missing extension-scoped Repomix guidance %q", path, want)
			}
		}
	}
	handoff := read("docs/agent-workflows/handoff-contract.md")
	for _, want := range []string{"Include Patterns", "Initial Output ID", "Review Output ID", "Pattern Match", "Scoped Fingerprint", "Stale"} {
		if !strings.Contains(handoff, want) {
			t.Errorf("handoff contract missing Repomix evidence field %q", want)
		}
	}
	architectureProfile := read("docs/agent-workflows/reviewer-profiles/architecture.md")
	for _, want := range []string{"same normalized extension-scoped `includePatterns`", "new baseline", "current source or gopls"} {
		if !strings.Contains(architectureProfile, want) {
			t.Errorf("architecture reviewer profile missing Repomix check %q", want)
		}
	}
	if !strings.Contains(read("docs/ai/agents-mcp-plugins-plan.md"), "已由目前的九個 child threads 政策取代") {
		t.Error("historical plan must identify its concurrency policy as superseded")
	}
	profiles := []string{"architecture", "api-contract", "security", "performance", "testing", "release", "incident"}
	for _, profile := range profiles {
		path := filepath.Join(root, "docs", "agent-workflows", "reviewer-profiles", profile+".md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("reviewer profile %s missing: %v", profile, err)
		}
	}
	if strings.Contains(string(mustRead(filepath.Join(root, ".codex", "config.toml"))), "reviewer-") {
		t.Error("reviewer profiles must not become Agent registrations")
	}
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}
