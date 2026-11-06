package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureAuditorContract(t *testing.T) {
	root := filepath.Join("..", "..")
	var a agent
	if err := readTOML(filepath.Join(root, ".codex", "agents", "architecture_auditor.toml"), &a); err != nil {
		t.Fatal(err)
	}
	if err := validateArchitectureAuditorInstructions(a.DeveloperInstructions); err != nil {
		t.Fatal(err)
	}
	if err := validateArchitectureAuditorWorkflow(root); err != nil {
		t.Fatal(err)
	}
}

func TestArchitectureAuditorInstructionContractRejectsRemovedClauses(t *testing.T) {
	var a agent
	if err := readTOML(filepath.Join("..", "..", ".codex", "agents", "architecture_auditor.toml"), &a); err != nil {
		t.Fatal(err)
	}
	assertArchitectureAuditorClauseMutationsFail(t, "instructions", a.DeveloperInstructions, architectureAuditorInstructionClauses)
}

func TestArchitectureAuditorWorkflowContractRejectsRemovedClauses(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "docs", "agent-workflows", "go-architecture-audit.md"))
	if err != nil {
		t.Fatal(err)
	}
	assertArchitectureAuditorClauseMutationsFail(t, "workflow", string(contents), architectureAuditorWorkflowClauses)
}

func assertArchitectureAuditorClauseMutationsFail(t *testing.T, kind, valid string, clauses []architectureAuditorClause) {
	t.Helper()
	for _, clause := range clauses {
		clause := clause
		t.Run(clause.name, func(t *testing.T) {
			mutated := strings.Replace(valid, clause.text, "", 1)
			if mutated == valid {
				t.Fatalf("test fixture does not contain clause %q", clause.text)
			}
			if err := validateArchitectureAuditorContractText(kind, mutated, clauses); err == nil {
				t.Fatalf("contract accepted removal of %s clause", clause.name)
			}
		})
	}
}
