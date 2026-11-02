package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const dummySHA = "abcdef0123456789abcdef0123456789abcdef01"

func completeFixture(now time.Time) evidence {
	e := evidence{
		SchemaVersion: 1, Environment: "staging",
		ProjectID: "linebot-acceptance-staging",
		CommitSHA: dummySHA, Reviewer: "test-reviewer",
		CheckedAt: now.UTC().Format(time.RFC3339),
		Checks:    make(map[string]checkEvidence),
	}
	for _, id := range requiredChecks {
		e.Checks[id] = checkEvidence{
			Status:    "passed",
			Reference: "https://github.com/963sup/linebot-platform/actions",
		}
	}
	return e
}

func TestAcceptanceGateRejectsIncompleteAndProduction(t *testing.T) {
	now := time.Now().UTC()
	if err := checkStagingEvidence(completeFixture(now), dummySHA, now); err != nil {
		t.Fatalf("synthetic gate fixture should validate: %v", err)
	}
	for _, tc := range []struct {
		name string
		edit func(*evidence)
	}{
		{"wrong environment", func(e *evidence) { e.Environment = "production" }},
		{"production project", func(e *evidence) { e.ProjectID = "linebot-production" }},
		{"missing reviewer", func(e *evidence) { e.Reviewer = "" }},
		{"old timestamp", func(e *evidence) { e.CheckedAt = now.Add(-31 * 24 * time.Hour).Format(time.RFC3339) }},
		{"missing check", func(e *evidence) { delete(e.Checks, "private_worker_iam") }},
		{"blocked check", func(e *evidence) {
			e.Checks["firestore_emulator"] = checkEvidence{Status: "blocked"}
		}},
		{"unauditable evidence", func(e *evidence) {
			e.Checks["ci_go"] = checkEvidence{Status: "passed", Reference: "http://example.test/?token=secret"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := completeFixture(now)
			tc.edit(&e)
			if err := checkStagingEvidence(e, dummySHA, now); err == nil {
				t.Fatal("unsafe or incomplete evidence was accepted")
			}
		})
	}
	if err := checkStagingEvidence(completeFixture(now), strings.Repeat("0", 40), now); err == nil {
		t.Fatal("evidence from a different commit was accepted")
	}
}

func TestLoadEvidenceDoesNotAcceptUnknownFieldsOrMultipleObjects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	fixture := completeFixture(time.Now())
	b, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEvidence(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, []byte("\n{}")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEvidence(path); err == nil {
		t.Fatal("multiple JSON objects must be rejected")
	}
}
