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
		"_LINE_MESSAGING_API_BOT_USER_ID",
		"--set-secrets=LINE_MESSAGING_API_CHANNEL_SECRET=line-messaging-api-channel-secret:latest",
		"--set-secrets=LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN=line-messaging-api-channel-access-token:latest",
		"--set-env-vars",
		"--clear-secrets",
		"--concurrency=16",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("cloudbuild.deploy.yaml missing %q", want)
		}
	}
	if strings.Count(text, "--concurrency=16") != 1 {
		t.Fatal("only the private worker deployment may set --concurrency=16")
	}
	if strings.Count(text, "--deploy-health-check") != 3 {
		t.Fatal("every Cloud Run candidate must pass its startup health check")
	}
	if strings.Count(text, "--ingress=all") != 2 || strings.Count(text, "--ingress=internal") != 1 {
		t.Fatal("public services must accept external ingress and only the worker may use internal ingress")
	}
	if strings.Contains(text, "_AR_REPOSITORY") {
		t.Fatal("deployment contains an unused Artifact Registry substitution")
	}
	for _, forbidden := range []string{"--update-env-vars", "--remove-env-vars", "--update-secrets", "--remove-secrets"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("deployment must use an exact allowlist instead of %q", forbidden)
		}
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
	for _, forbidden := range []string{"LINE_EVENT_MODE", "LINE_API_TIMEOUT", "LOG_LEVEL", "LOG_FORMAT", "LINE_MESSAGING_API_BOT_USER_ID", "LINE_MINI_APP_LIFF_URL", "FIRESTORE_DATABASE"} {
		if strings.Contains(privateStep, forbidden) {
			t.Errorf("private worker deployment contains unused setting %q", forbidden)
		}
	}
	miniapp := strings.Index(text, "id: 'deploy-miniapp'")
	if miniapp < 0 {
		t.Fatal("MINI App deployment step is missing")
	}
	miniappStep := text[miniapp:]
	if next := strings.Index(miniappStep[1:], "\n  - id:"); next >= 0 {
		miniappStep = miniappStep[:next+1]
	}
	for _, want := range []string{"--service-account=${_MINIAPP_SERVICE_ACCOUNT}", "--allow-unauthenticated", "LINE_MINI_APP_CHANNEL_ID=${_LINE_MINI_APP_CHANNEL_ID}", "LINE_MINI_APP_LIFF_URL=${_LINE_MINI_APP_LIFF_URL}", "LINE_MINI_APP_ENDPOINT_URL=${_LINE_MINI_APP_ENDPOINT_URL}", "--clear-secrets"} {
		if !strings.Contains(miniappStep, want) {
			t.Errorf("MINI App deployment missing %q", want)
		}
	}
	for _, forbidden := range []string{"LINE_MESSAGING_API_CHANNEL_SECRET=line-messaging-api-channel-secret", "LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN=line-messaging-api-channel-access-token", "LINE_API_TIMEOUT", "LOG_LEVEL", "LOG_FORMAT", "FIRESTORE_DATABASE", "--concurrency=16"} {
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
