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
	buildContents, err := os.ReadFile("../../cloudbuild.yaml")
	if err != nil {
		t.Fatal(err)
	}
	buildText := string(buildContents)
	for _, want := range []string{
		"go build -trimpath -o /workspace/.release/miniapp ./cmd/miniapp",
		"'BINARY=miniapp'",
		"/line-miniapp:$BUILD_ID",
	} {
		if !strings.Contains(buildText, want) {
			t.Errorf("cloudbuild.yaml missing MINI App build contract %q", want)
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
		"line-miniapp",
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
	miniapp := strings.Index(text, "id: 'deploy-miniapp'")
	if miniapp < 0 {
		t.Fatal("MINI App deployment step is missing")
	}
	miniappStep := text[miniapp:]
	if next := strings.Index(miniappStep[1:], "\n  - id:"); next >= 0 {
		miniappStep = miniappStep[:next+1]
	}
	for _, want := range []string{"--service-account=${_MINIAPP_SERVICE_ACCOUNT}", "--allow-unauthenticated", "LINE_LIFF_ID=${_LINE_LIFF_ID}", "LINE_LOGIN_CHANNEL_ID=${_LINE_LOGIN_CHANNEL_ID}", "MINIAPP_PUBLIC_ORIGIN=${_MINIAPP_PUBLIC_ORIGIN}", "--remove-env-vars=GCP_PROJECT_ID,ATTENDANCE_LOCATION_ACTIVE_KEY_ID,ATTENDANCE_LOCATION_KEY_VERSIONS"} {
		if !strings.Contains(miniappStep, want) {
			t.Errorf("MINI App deployment missing %q", want)
		}
	}
	for _, forbidden := range []string{"LINE_CHANNEL_SECRET=line-bot-channel-secret", "LINE_CHANNEL_ACCESS_TOKEN=line-bot-channel-access-token", "--concurrency=16"} {
		if strings.Contains(miniappStep, forbidden) {
			t.Errorf("MINI App deployment contains forbidden setting %q", forbidden)
		}
	}
	workerSource, err := os.ReadFile("../../internal/bootstrap/worker.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workerSource), "workerMaxConcurrentRequests = 16") {
		t.Fatal("Cloud Run concurrency must match the worker admission cap")
	}
}
