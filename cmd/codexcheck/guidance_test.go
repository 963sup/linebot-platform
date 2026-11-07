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
	if got := roles["architecture_auditor"].Description; !strings.Contains(got, "semantics, abstractions, types, interfaces, functions, naming") {
		t.Errorf("architecture_auditor registry description does not route semantic and naming audits: %q", got)
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
	audit := read("docs/agent-workflows/go-architecture-audit.md")
	for _, marker := range []string{
		"includePatterns: **/*.go", "ignorePatterns: **/*_test.go", "docs/assets/architecture/**",
		"No code or architecture asset edits before", "one decision-complete plan", "same-pattern Repomix MCP review pack",
		"語意、抽象、型別、介面、package、函數與命名不一致", "Keep`、`Rename`、`Merge`、`Delete` 或 `Exception",
		"Mandatory architecture decision matrix", "Aggregate Root", "Domain Error", "Application Service",
		"Input Port", "Output Port", "Unit of Work", "Transaction Boundary", "Event Handler", "Authorization",
		"Repository implementations are outbound adapters", "Anti-Corruption Layer", "not a scaffold checklist",
		"production consumer", "Not Applicable", "internal/application/usecases/<capability>/",
		"cmd/*", "internal/bootstrap", "Candidate Contexts are not promoted", "Authority order",
	} {
		if !strings.Contains(audit, marker) {
			t.Errorf("full-read audit missing %q", marker)
		}
	}
	method := read(".agents/skills/repository-engineering-method/SKILL.md")
	for _, marker := range []string{
		"Classify scope first", "Localized", "Cross-package / external contract", "Repository-wide architecture",
		"samber/cc-skills-golang", "samber/cc-skills", "Context7", "provider MCP", "Agents/subagents",
		"DDD", "Hexagonal Architecture", "Go Idioms", "KISS", "YAGNI", "DRY",
		"pinned Mise", "unavailable", "Not Applicable",
	} {
		if !strings.Contains(method, marker) {
			t.Errorf("repository engineering method missing %q", marker)
		}
	}
	rootGuidance := read("AGENTS.md")
	for _, marker := range []string{
		"repository-engineering-method/SKILL.md", "Repomix inventories", "Graphify explores",
		"samber/cc-skills-golang", "samber/cc-skills", "Context7 and provider MCPs",
		"DDD, Hexagonal Architecture, Go Idioms, KISS, YAGNI and DRY", "pinned Mise tasks", "bootstrap.sh exec -- gofmt",
	} {
		if !strings.Contains(rootGuidance, marker) {
			t.Errorf("AGENTS.md missing repository method contract %q", marker)
		}
	}
	mise := read("mise.toml")
	for _, marker := range []string{"BOOTSTRAP_WORKSPACE", "BOOTSTRAP_REPO", "mise run plugins-install", "mise run plugins-install-standard", "mise run graphify-update", "mise run agent-preflight"} {
		if !strings.Contains(mise, marker) {
			t.Errorf("mise setup missing autonomous preparation step %q", marker)
		}
	}
	if !strings.Contains(mise, "graphify extract ${BOOTSTRAP_REPO} --code-only --allow-dedup-shrink") {
		t.Error("graphify update must accept safe dedup shrink in its rebuildable cache")
	}
	blueprint := read("docs/architecture/blueprint.md")
	for _, marker := range []string{
		"Business Fact -> Invariant -> Owner -> Boundary -> Component Kind -> Name -> Acceptance Test",
		"Structure = Facts + Invariants + Ownership + Dependencies",
		"Abstraction = Necessary Behavior + Real Boundary + Production Consumer",
		"Name = shortest unambiguous official term within its Context",
		"`term`、`definition`、`forbidden_aliases` 與 `code_symbols` 是 Semantic Registry",
		"Existing Symbol -> Semantic Registry / Capability Ownership -> Keep / Rename / Merge / Delete / Exception -> gopls Impact -> Ordered Migration -> Tests -> Repomix Review",
		"不得留下 unresolved／unmapped core item",
		"長期例外必須由 accepted ADR 核准",
	} {
		if !strings.Contains(blueprint, marker) {
			t.Errorf("architecture blueprint missing semantic derivation contract %q", marker)
		}
	}
	for _, path := range []string{"AGENTS.md", "docs/ai/agents.md", "docs/agent-workflows/routing-policy.md"} {
		guidance := read(path)
		if !strings.Contains(guidance, "architecture_auditor") {
			t.Errorf("%s does not route repository-wide audits", path)
		}
		if path == "docs/agent-workflows/routing-policy.md" && !strings.Contains(guidance, "mandatory six-part decision matrix") {
			t.Errorf("%s does not require the architecture decision matrix before writers", path)
		}
	}
	if !strings.Contains(read("AGENTS.md"), "semantic/naming unification") {
		t.Error("AGENTS.md does not route repository-wide semantic and naming unification")
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
	for _, want := range []string{"9 個角色", "15 宣告、10 啟用、5 個服務維持停用", "1／4／8／9", "0／10"} {
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
	for _, want := range []string{"Repomix Lifecycle", "Include Patterns", "Baseline Output ID", "Review Output ID or Not Applicable", "Pattern Match or Not Applicable", "Scoped Fingerprint", "Stale", "Method Scope", "Design Lenses", "Graphify Evidence", "External Documentation"} {
		if !strings.Contains(handoff, want) {
			t.Errorf("handoff contract missing Repomix evidence field %q", want)
		}
	}
	architectureProfile := read("docs/agent-workflows/reviewer-profiles/architecture.md")
	for _, want := range []string{"same normalized extension-scoped `includePatterns`", "new baseline", "current source or gopls", "without replacing", "Not Applicable"} {
		if !strings.Contains(architectureProfile, want) {
			t.Errorf("architecture reviewer profile missing Repomix check %q", want)
		}
	}
	if !strings.Contains(read("docs/ai/agents-mcp-plugins-plan.md"), "已由目前的九個 child threads 政策取代") {
		t.Error("historical plan must identify its concurrency policy as superseded")
	}
	for _, path := range []string{"docs/roadmap.md", "docs/ai/agents-mcp-plugins-plan.md"} {
		if strings.Contains(read(path), "現行 8 個角色") || strings.Contains(read(path), "八個 depth-one specialist roles") {
			t.Errorf("%s describes a stale current role count", path)
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
