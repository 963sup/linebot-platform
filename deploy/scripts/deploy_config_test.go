package scripts_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestCloudBuildDeploymentSafetyContract(t *testing.T) {
	raw, err := os.ReadFile("../../cloudbuild.deploy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatalf("parse cloudbuild.deploy.yaml: %v", err)
	}
	text := string(raw)
	for _, required := range []string{
		"_APPROVE_DEPLOYMENT: 'false'",
		"_STAGING_EVIDENCE_SHA256: ''",
		"_TASK_WORKER_ORIGIN: ''",
		"--service-account=${_PUBLIC_SERVICE_ACCOUNT}",
		"--service-account=${_WORKER_SERVICE_ACCOUNT}",
		"--service-account=${_MINIAPP_SERVICE_ACCOUNT}",
		"_MINIAPP_IMAGE_URI: ''",
		"_LINE_MINI_APP_CHANNEL_ID: ''",
		"_LINE_MINI_APP_LIFF_URL: ''",
		"LINE_MINI_APP_LIFF_URL=${_LINE_MINI_APP_LIFF_URL}",
		"_LINE_MINI_APP_ENDPOINT_URL: ''",
		"_ATTENDANCE_LOCATION_ACTIVE_KEY_ID: ''",
		"_ATTENDANCE_LOCATION_KEY_VERSIONS: ''",
		"--ingress=internal",
		"--concurrency=16",
		"--no-traffic",
		"--format=value(status.latestCreatedRevisionName)",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("deployment contract missing %q", required)
		}
	}
	if strings.Contains(text, "--service-account=line-bot-runtime@") {
		t.Error("public and worker must not share the legacy runtime identity")
	}
	if strings.Count(text, "LINE_MINI_APP_LIFF_URL=${_LINE_MINI_APP_LIFF_URL}") != 2 {
		t.Error("Attendance entry must be derived once for the public webhook only")
	}
	if strings.Contains(text, "update-traffic") || strings.Contains(text, "latestReadyRevisionName") {
		t.Error("candidate deployment must not promote or infer a revision")
	}
}

func TestPreflightBindsMiniAppConfigToApprovedEvidence(t *testing.T) {
	raw, err := os.ReadFile("preflight.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, required := range []string{
		`evidence_miniapp_endpoint_url="$(jq -r '.miniapp_config.endpoint_url'`,
		`evidence_miniapp_liff_url="$(jq -r '.miniapp_config.liff_url'`,
		`evidence_miniapp_channel_id="$(jq -r '.miniapp_config.channel_id'`,
		`"${evidence_miniapp_endpoint_url}" != "${LINE_MINI_APP_ENDPOINT_URL}"`,
		`"${evidence_miniapp_liff_url}" != "${LINE_MINI_APP_LIFF_URL}"`,
		`"${evidence_miniapp_channel_id}" != "${LINE_MINI_APP_CHANNEL_ID}"`,
		`"${evidence_attendance_key_id}" != "${ATTENDANCE_LOCATION_ACTIVE_KEY_ID}"`,
	} {
		if !strings.Contains(text, required) {
			t.Errorf("preflight does not bind approved MINI App configuration: missing %q", required)
		}
	}

	evidenceRaw, err := os.ReadFile("../evidence/staging.example.json")
	if err != nil {
		t.Fatal(err)
	}
	var evidence struct {
		MiniAppConfig struct {
			EndpointURL string `json:"endpoint_url"`
			LIFFURL     string `json:"liff_url"`
			ChannelID   string `json:"channel_id"`
			LocationKeyID  string `json:"attendance_location_active_key_id"`
			KeyMappingHash string `json:"attendance_location_key_versions_sha256"`
		} `json:"miniapp_config"`
	}
	if err := json.Unmarshal(evidenceRaw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.MiniAppConfig.EndpointURL == "" || evidence.MiniAppConfig.LIFFURL == "" || evidence.MiniAppConfig.ChannelID == "" || evidence.MiniAppConfig.LocationKeyID == "" || evidence.MiniAppConfig.KeyMappingHash == "" {
		t.Fatalf("staging evidence omits approved MINI App configuration: %+v", evidence.MiniAppConfig)
	}
}

func TestPromotionScopeExpiryAndCommandFailure(t *testing.T) {
	tests := []struct {
		name       string
		expires    string
		service    string
		failUpdate bool
		wantOK     bool
		wantCalls  bool
	}{
		{name: "valid single service", expires: "2099-01-01T00:00:00Z", service: "line-bot-private", wantOK: true, wantCalls: true},
		{name: "expired", expires: "2000-01-01T00:00:00Z", service: "line-bot-private"},
		{name: "scope mismatch", expires: "2099-01-01T00:00:00Z", service: "another-service"},
		{name: "gcloud failure", expires: "2099-01-01T00:00:00Z", service: "line-bot-private", failUpdate: true, wantCalls: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "gcloud.log")
			fake := filepath.Join(dir, "gcloud")
			fakeBody := "#!/usr/bin/env bash\nset -eu\necho \"$*\" >>\"$GCLOUD_LOG\"\nif [[ \"${FAIL_UPDATE:-}\" == 1 && \"$*\" == *\"update-traffic\"* ]]; then exit 17; fi\n"
			if err := os.WriteFile(fake, []byte(fakeBody), 0o700); err != nil {
				t.Fatal(err)
			}
			evidence := map[string]any{
				"schema_version": 1, "project_id": "test-project", "region": "asia-east1",
				"service": tt.service, "candidate_revision": "candidate-00002", "base_revision": "base-00001",
				"approved_percent": 10, "observed_at_utc": "2026-11-05T00:00:00Z", "expires_at_utc": tt.expires,
				"reviewer": "reviewer", "approved": true, "smoke_passed": true,
			}
			raw, err := json.Marshal(evidence)
			if err != nil {
				t.Fatal(err)
			}
			evidencePath := filepath.Join(dir, "evidence.json")
			if err := os.WriteFile(evidencePath, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(raw)
			cmd := exec.Command("bash", "promote.sh")
			cmd.Env = append(os.Environ(),
				"PATH="+dir+":"+os.Getenv("PATH"), "GCLOUD_LOG="+logPath,
				"PROJECT_ID=test-project", "REGION=asia-east1", "SERVICE=line-bot-private",
				"CANDIDATE_REVISION=candidate-00002", "BASE_REVISION=base-00001", "TARGET_PERCENT=10",
				"PROMOTION_EVIDENCE_FILE="+evidencePath, "PROMOTION_EVIDENCE_SHA256="+hex.EncodeToString(digest[:]),
			)
			if tt.failUpdate {
				cmd.Env = append(cmd.Env, "FAIL_UPDATE=1")
			}
			err = cmd.Run()
			if (err == nil) != tt.wantOK {
				t.Fatalf("promotion error = %v, wantOK=%v", err, tt.wantOK)
			}
			if tt.failUpdate {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != 17 {
					t.Fatalf("command failure = %v, want propagated exit 17", err)
				}
			}
			calls, readErr := os.ReadFile(logPath)
			hadCalls := readErr == nil && len(calls) > 0
			if hadCalls != tt.wantCalls {
				t.Fatalf("gcloud calls=%q, wantCalls=%v", calls, tt.wantCalls)
			}
			if tt.wantOK {
				text := string(calls)
				if strings.Count(text, "services update-traffic") != 1 || !strings.Contains(text, "--to-revisions=candidate-00002=10,base-00001=90") {
					t.Fatalf("unexpected promotion calls: %s", text)
				}
			}
			if tt.failUpdate && !strings.Contains(string(calls), "services update-traffic") {
				t.Fatalf("failure did not originate from update command: %s", calls)
			}
		})
	}
}

func TestTerraformConsumesAuthoritativeFirestoreContract(t *testing.T) {
	raw, err := os.ReadFile("../firestore/indexes.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Indexes        []any `json:"indexes"`
		FieldOverrides []struct {
			CollectionGroup string `json:"collectionGroup"`
			FieldPath       string `json:"fieldPath"`
			TTL             bool   `json:"ttl"`
		} `json:"fieldOverrides"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.Indexes) == 0 {
		t.Fatalf("unexpected authoritative Firestore contract: indexes=%d overrides=%+v", len(contract.Indexes), contract.FieldOverrides)
	}
	wantTTL := map[string]bool{
		"line_inbox_v1.expires_at":      true,
		"line_outbox_v1.expires_at":     true,
		"identity_outbox_v1.expires_at": true,
	}
	for _, override := range contract.FieldOverrides {
		if override.TTL {
			delete(wantTTL, override.CollectionGroup+"."+override.FieldPath)
		}
	}
	if len(wantTTL) != 0 {
		t.Fatalf("authoritative Firestore contract is missing TTL policies: %v", wantTTL)
	}
	tf, err := os.ReadFile("../terraform/firestore.tf")
	if err != nil {
		t.Fatal(err)
	}
	text := string(tf)
	for _, required := range []string{"jsondecode(file(\"${path.module}/../firestore/indexes.json\"))", "local.firestore_contract.indexes", "local.firestore_contract.fieldOverrides", "if try(override.ttl, false)"} {
		if !strings.Contains(text, required) {
			t.Errorf("Terraform Firestore contract missing %q", required)
		}
	}
	for _, forbidden := range []string{"line_inbox_v1", "line_outbox_v1", "identity_outbox_v1"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("Terraform must not hand-maintain collection %q", forbidden)
		}
	}
}

func TestTerraformGrantsMiniAppOnlyAttendancePersistenceSecrets(t *testing.T) {
	raw, err := os.ReadFile("../terraform/main.tf")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, required := range []string{
		`resource "google_project_iam_member" "miniapp_datastore"`,
		`member  = "serviceAccount:${google_service_account.runtime["miniapp"].email}"`,
		`data "google_secret_manager_secret" "attendance_location_key"`,
		`resource "google_secret_manager_secret_iam_member" "miniapp_attendance_location_key"`,
	} {
		if !strings.Contains(text, required) {
			t.Errorf("Terraform MINI App Attendance contract missing %q", required)
		}
	}
	for _, forbidden := range []string{
		`miniapp_channel_secret`, `miniapp_access_token`, `miniapp_tasks`,
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("Terraform grants forbidden MINI App capability %q", forbidden)
		}
	}
}
