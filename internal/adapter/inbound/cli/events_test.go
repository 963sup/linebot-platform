package cli

import (
	"strings"
	"testing"
)

func TestEventsRequiresExplicitReadOnlyModeAndStagingAllowlist(t *testing.T) {
	approved := func(key string) string {
		if key == "LINEBOT_STAGING_PROJECT" {
			return "test-staging"
		}
		return ""
	}
	for _, tc := range []struct {
		name string
		args []string
		env  func(string) string
		want string
	}{
		{"missing permission", []string{"--project", "test-staging"}, approved, "choose exactly one"},
		{"both modes", []string{"--project", "test-staging", "--inspect", "--check"}, approved, "choose exactly one"},
		{"wrong project", []string{"--project", "prod", "--inspect"}, approved, "LINEBOT_STAGING_PROJECT"},
		{"missing allowlist", []string{"--project", "test-staging", "--inspect"}, func(string) string { return "" }, "LINEBOT_STAGING_PROJECT"},
		{"invalid state", []string{"--project", "test-staging", "--inspect", "--state", "broken"}, approved, "invalid outbox"},
		{"oversize", []string{"--project", "test-staging", "--inspect", "--limit", "200"}, approved, "invalid outbox"},
		{"invalid threshold", []string{"--project", "test-staging", "--check", "--max-pending-age", "0s"}, approved, "invalid outbox"},
		{"unknown flag", []string{"--project", "test-staging", "--inspect", "--send"}, approved, "flag provided but not defined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runEvents(testContext(), tc.args, tc.env)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q error, got %v", tc.want, err)
			}
		})
	}
}
