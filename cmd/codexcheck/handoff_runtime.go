package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateHandoffRuntime(root string) error {
	read := func(path string) (string, error) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		return string(b), err
	}
	mise, err := read("mise.toml")
	if err != nil {
		return err
	}
	for _, marker := range []string{
		`[tasks.handoff-snapshot]`, `go run ./cmd/codexhandoff snapshot`,
		`[tasks.handoff-check]`, `go run ./cmd/codexhandoff validate`,
		`[tasks.agent-preflight]`, `[tasks.plugins-check]`,
		`go run ./cmd/codexplugins --install`,
		`does not prove host or Agent exposure`,
	} {
		if !strings.Contains(mise, marker) {
			return fmt.Errorf("mise.toml missing approved handoff/plugin marker %q", marker)
		}
	}
	for _, forbidden := range []string{
		"codex plugin add agent-skills@agent-skills",
		"codex plugin marketplace add addyosmani/agent-skills",
		"codex plugin marketplace add google/skills",
	} {
		if strings.Contains(mise, forbidden) {
			return fmt.Errorf("mise.toml contains unpinned plugin command %q", forbidden)
		}
	}
	handoff, err := read("docs/agent-workflows/handoff-contract.md")
	if err != nil {
		return err
	}
	for _, marker := range []string{"PREPARE", "READY", "START", "RESULT", "native-verified", "explicit", "Handoff ID", "Tool Evidence", "Skill Evidence"} {
		if !strings.Contains(handoff, marker) {
			return fmt.Errorf("handoff contract missing %q", marker)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, ".codex", "agents"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		body, err := read(filepath.ToSlash(filepath.Join(".codex", "agents", entry.Name())))
		if err != nil {
			return err
		}
		if !strings.Contains(body, "handoff-contract.md") || !strings.Contains(body, "READY") {
			return fmt.Errorf("agent %s must require the Handoff v1 READY gate", entry.Name())
		}
	}
	return nil
}
