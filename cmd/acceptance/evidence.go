// Package main validates reviewer-supplied staging evidence. It never
// connects to external APIs or treats unit tests as live staging proof.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"
)

type evidence struct {
	SchemaVersion int                      `json:"schema_version"`
	Environment   string                   `json:"environment"`
	ProjectID     string                   `json:"project_id"`
	CommitSHA     string                   `json:"commit_sha"`
	Reviewer      string                   `json:"reviewer"`
	CheckedAt     string                   `json:"checked_at"`
	Checks        map[string]checkEvidence `json:"checks"`
}

type checkEvidence struct {
	Status    string `json:"status"`
	Reference string `json:"reference"`
}

var requiredChecks = []string{
	"ci_go",
	"firestore_emulator",
	"line_redelivery",
	"cloud_tasks_oidc",
	"private_worker_iam",
	"firestore_ttl",
	"scheduled_reconcile",
	"unknown_outcome",
	"privacy_logging",
	"codex_line_rich_menu",
	"codex_google_tasks",
	"codex_google_calendar",
	"codex_ai_reply",
	"codex_durable_redelivery",
	"codex_sdk_upgrade",
}

// checkStagingEvidence is an administrative evidence gate, not an external
// service probe. The reviewer must actually inspect every referenced artifact.
func checkStagingEvidence(e evidence, headSHA string, now time.Time) error {
	if e.SchemaVersion != 1 || e.Environment != "staging" {
		return errors.New("stage-5 evidence must use schema_version=1 and environment=staging")
	}
	if len(e.ProjectID) < 9 || !strings.Contains(strings.ToLower(e.ProjectID), "staging") ||
		strings.Contains(strings.ToLower(e.ProjectID), "production") || strings.Contains(strings.ToLower(e.ProjectID), "-prod") {
		return errors.New("stage-5 evidence must identify a dedicated non-production staging project")
	}
	if len(e.CommitSHA) != 40 || e.CommitSHA != headSHA {
		return errors.New("stage-5 evidence must reference exactly the current repository HEAD")
	}
	if len(strings.TrimSpace(e.Reviewer)) < 3 {
		return errors.New("stage-5 evidence requires a named reviewer")
	}
	checked, err := time.Parse(time.RFC3339, e.CheckedAt)
	if err != nil || checked.After(now.Add(5*time.Minute)) || checked.Before(now.Add(-30*24*time.Hour)) {
		return errors.New("stage-5 evidence timestamp must be recent RFC3339")
	}
	if len(e.Checks) != len(requiredChecks) {
		return fmt.Errorf("stage-5 evidence must include exactly %d required checks", len(requiredChecks))
	}
	for name := range e.Checks {
		if !slices.Contains(requiredChecks, name) {
			return fmt.Errorf("unexpected stage-5 evidence check %s", name)
		}
	}
	for _, name := range requiredChecks {
		check, exists := e.Checks[name]
		if !exists || check.Status != "passed" {
			return fmt.Errorf("stage-5 acceptance not complete: %s", name)
		}
		link, err := url.Parse(check.Reference)
		if err != nil || link.Scheme != "https" || link.Host == "" ||
			link.User != nil || link.RawQuery != "" || link.Fragment != "" {
			return fmt.Errorf("stage-5 check %s needs an HTTPS evidence URL without credentials or query parameters", name)
		}
	}
	return nil
}

func loadEvidence(filename string) (evidence, error) {
	f, err := os.Open(filename)
	if err != nil {
		return evidence{}, err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 128*1024))
	d.DisallowUnknownFields()
	var e evidence
	if err := d.Decode(&e); err != nil {
		return evidence{}, err
	}
	var extra interface{}
	if err := d.Decode(&extra); err != io.EOF {
		return evidence{}, errors.New("stage-5 evidence must contain exactly one JSON object")
	}
	return e, nil
}
