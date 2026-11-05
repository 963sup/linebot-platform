package contract

import (
	"os"
	"strings"
	"testing"
)

func TestCloudBuildConfigurationTargetsLineBotProduction(t *testing.T) {
	for _, file := range []string{"cloudbuild.yaml", "cloudbuild.deploy.yaml"} {
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
		"_LINE_BOT_DESTINATION",
		"LINE_CHANNEL_SECRET=line-bot-channel-secret:latest",
		"LINE_CHANNEL_ACCESS_TOKEN=line-bot-channel-access-token:latest",
		"--remove-env-vars",
		"--remove-secrets",
		"--concurrency=16",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("cloudbuild.deploy.yaml missing %q", want)
		}
	}
	if strings.Count(text, "--concurrency=16") != 1 {
		t.Fatal("only the private worker deployment may set --concurrency=16")
	}
	private := strings.Index(text, "id: 'deploy-private-worker'")
	if private < 0 {
		t.Fatal("private worker deployment step is missing")
	}
	privateStep := text[private:]
	if next := strings.Index(privateStep[1:], "\n  - id:"); next >= 0 {
		privateStep = privateStep[:next+1]
	}
	if !strings.Contains(privateStep, "--concurrency=16") {
		t.Fatal("private worker deployment must own the concurrency limit")
	}
	workerSource, err := os.ReadFile("../../internal/bootstrap/worker.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workerSource), "workerMaxConcurrentRequests = 16") {
		t.Fatal("Cloud Run concurrency must match the worker admission cap")
	}
}
