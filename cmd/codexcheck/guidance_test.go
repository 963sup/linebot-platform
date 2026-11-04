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
		state := "停用"
		if server.Enabled {
			state = "啟用"
		}
		if !strings.Contains(mcpDoc, "| `"+name+"` | "+state+" |") {
			t.Errorf("MCP %s state missing or stale", name)
		}
	}
	for _, path := range []string{"AGENTS.md", "docs/ai/agents.md", ".codex/agents/architect.toml"} {
		if strings.Contains(read(path), "DIRECTORY-STRUCTURE.md") {
			t.Errorf("%s references deleted architecture target", path)
		}
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
