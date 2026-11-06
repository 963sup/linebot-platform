package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisteredAgentsDeclareRoleSpecificEvidenceLifecycle(t *testing.T) {
	root := filepath.Join("..", "..")
	for role, lifecycle := range approvedAgentLifecycles {
		t.Run(role, func(t *testing.T) {
			var a agent
			if err := readTOML(filepath.Join(root, ".codex", "agents", role+".toml"), &a); err != nil {
				t.Fatal(err)
			}
			if err := validateAgentEvidence(role, a); err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(a.DeveloperInstructions, "Lifecycle: "+lifecycle+"."); got != 1 {
				t.Fatalf("lifecycle marker count = %d, want 1", got)
			}
		})
	}
}

func TestArchitectureAuditorIsReadOnlyAndRequiresCompleteSnapshot(t *testing.T) {
	var a agent
	if err := readTOML(filepath.Join("..", "..", ".codex", "agents", "architecture_auditor.toml"), &a); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"Lifecycle: read-only.", "**/*.go", "**/*_test.go", "docs/assets/architecture/**", "Do not edit while reading", "do not spawn subagents"} {
		if !strings.Contains(a.DeveloperInstructions, marker) {
			t.Errorf("auditor instructions missing %q", marker)
		}
	}
}

func TestAgentEvidenceRejectsWrongOrUniversalLifecycle(t *testing.T) {
	validWriter := agent{DeveloperInstructions: "Lifecycle: writer. Follow " + agentHandoffContract + "; consume the orchestrator-owned baseline and provide review evidence."}
	tests := []struct {
		name  string
		role  string
		agent agent
	}{
		{name: "wrong role lifecycle", role: "researcher", agent: validWriter},
		{name: "missing contract", role: "application", agent: agent{DeveloperInstructions: "Lifecycle: writer. consume the orchestrator-owned baseline and provide review evidence."}},
		{name: "duplicate lifecycle", role: "application", agent: agent{DeveloperInstructions: validWriter.DeveloperInstructions + " Lifecycle: writer."}},
		{name: "obsolete universal clause", role: "application", agent: agent{DeveloperInstructions: validWriter.DeveloperInstructions + " same-scope post-change pack"}},
		{name: "reviewer replaces evidence", role: "reviewer", agent: agent{DeveloperInstructions: "Lifecycle: reviewer. Follow " + agentHandoffContract + "; create a replacement pair and mark read-only Not Applicable."}},
		{name: "read-only requires pair", role: "researcher", agent: agent{DeveloperInstructions: "Lifecycle: read-only. Follow " + agentHandoffContract + "; always create two packs."}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateAgentEvidence(tc.role, tc.agent); err == nil {
				t.Fatal("invalid evidence lifecycle accepted")
			}
		})
	}
}
