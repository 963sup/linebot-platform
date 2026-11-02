package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type scenarioCatalog struct {
	SchemaVersion int            `toml:"schema_version"`
	Scenarios     []codexScenario `toml:"scenarios"`
}

type codexScenario struct {
	ID        string   `toml:"id"`
	Prompt    string   `toml:"prompt"`
	Module    string   `toml:"module"`
	Reference string   `toml:"reference"`
	Checks    []string `toml:"checks"`
}

// validateScenarios ensures a fresh Codex task has a grounded, versioned
// provider reference and review criteria. It does not execute Codex and does
// not claim any real cloud service or agent-session acceptance has passed.
func validateScenarios(root string) error {
	var catalog scenarioCatalog
	if err := readTOML(filepath.Join(root, "docs", "ai", "zero-context-scenarios.toml"), &catalog); err != nil {
		return err
	}
	if catalog.SchemaVersion != 1 {
		return fmt.Errorf("Codex scenario schema version must be 1")
	}
	required := []string{"line-rich-menu", "google-tasks", "google-calendar", "ai-reply", "durable-redelivery", "sdk-upgrade"}
	modules, err := directModules(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, s := range catalog.Scenarios {
		if seen[s.ID] || s.ID == "" {
			return fmt.Errorf("duplicate or empty Codex scenario ID %q", s.ID)
		}
		if !slices.Contains(required, s.ID) {
			return fmt.Errorf("unexpected Codex acceptance scenario %q", s.ID)
		}
		seen[s.ID] = true
		if _, ok := modules[s.Module]; !ok {
			return fmt.Errorf("scenario %s uses a module not directly declared in go.mod", s.ID)
		}
		if len(strings.TrimSpace(s.Prompt)) < 10 || len(s.Checks) < 3 {
			return fmt.Errorf("scenario %s needs a clear natural-language task and at least three review checks", s.ID)
		}
		for _, check := range s.Checks {
			if strings.TrimSpace(check) == "" {
				return fmt.Errorf("scenario %s contains an empty review criterion", s.ID)
			}
		}
		clean := filepath.Clean(s.Reference)
		if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) ||
			clean == ".." || clean != s.Reference {
			return fmt.Errorf("scenario %s has an unsafe reference path", s.ID)
		}
		info, err := os.Stat(filepath.Join(root, clean))
		if err != nil || info.IsDir() {
			return fmt.Errorf("scenario %s references missing source %s", s.ID, s.Reference)
		}
	}
	if len(seen) != len(required) {
		return fmt.Errorf("Codex scenario coverage incomplete: %d/%d", len(seen), len(required))
	}
	return nil
}
