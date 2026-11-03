package contract

import (
	"os"
	"strings"
	"testing"
)

func TestCloudBuildConfigurationTargetsLineBotProduction(t *testing.T) {
	for _, file := range []string{"cloudbuild.yaml", "cloudbuild.worker.yaml", "cloudbuild.deploy.yaml"} {
		contents, err := os.ReadFile("../../" + file)
		if err != nil {
			t.Fatal(err)
		}
		text := string(contents)
		if !strings.Contains(text, "line-bot") || strings.Contains(text, "linebot-platform") {
			t.Fatalf("%s must target the line-bot Artifact Registry repository", file)
		}
	}

	contents, err := os.ReadFile("../../cloudbuild.deploy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	for _, want := range []string{
		"bubbly-area-513305-s0",
		"line-bot-public",
		"line-bot-private",
		"line-bot-events",
		"U31469a4d9b5533208c71199a43b47803",
		"LINE_CHANNEL_SECRET=line-bot-channel-secret:latest",
		"LINE_CHANNEL_ACCESS_TOKEN=line-bot-channel-access-token:latest",
		"--remove-env-vars",
		"--remove-secrets",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("cloudbuild.deploy.yaml missing %q", want)
		}
	}
}
