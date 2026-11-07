package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func TestGeneratedDocumentationAndExamples(t *testing.T) {
	root := filepath.Join("..", "..")
	doc, err := os.ReadFile(filepath.Join(root, documentPath))
	if err != nil {
		t.Fatal(err)
	}
	if string(doc) != bootstrap.ConfigurationMarkdown() {
		t.Fatal("configuration documentation drifted from Go contract")
	}
	example, err := os.ReadFile(filepath.Join(root, examplePath))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkExample(example); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateRequiresOnlyKnownSettingsAndNoSecrets(t *testing.T) {
	for _, example := range []string{
		"UNEXPECTED=1\n",
		"LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN=private-token\n",
		"LINE_MESSAGING_API_CHANNEL_SECRET=private-secret\n",
		"APP_ENV=local\nPORT=8080\n",
	} {
		if err := checkExample([]byte(example)); err == nil {
			t.Fatalf("expected unsafe/incomplete example to be rejected")
		}
	}
}

func TestValidatePrintsNoSecretValues(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"--validate", "--role", "webhook"}, func(string) string { return "" }, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "role=webhook") || !strings.Contains(out.String(), "profile=local") {
		t.Fatalf("unsafe or missing safe metadata: %q", out.String())
	}
	for _, args := range [][]string{{}, {"--validate", "--check"}, {"--validate", "--role", "invalid"}} {
		if err := run(args, func(string) string { return "" }, &out); err == nil {
			t.Fatalf("invalid action %v was accepted", args)
		}
	}
}

func TestValidateMiniApp(t *testing.T) {
	values := map[string]string{
		"LINE_MINI_APP_CHANNEL_ID": "2000000000", "LINE_MINI_APP_LIFF_URL": "https://miniapp.line.me/2000000000-example",
		"LINE_MINI_APP_ENDPOINT_URL": "http://localhost:8080/miniapp/",
	}
	var out bytes.Buffer
	if err := run([]string{"--validate", "--role", "miniapp"}, func(key string) string { return values[key] }, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "role=miniapp") || !strings.Contains(out.String(), "mode=n/a") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestExampleRejectsInvalidRuntimeValues(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("..", "..", examplePath))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, before, after string
	}{
		{"invalid port", "PORT=8080", "PORT=0"},
		{"invalid timeout", "LINE_API_TIMEOUT=8s", "LINE_API_TIMEOUT=0s"},
		{"invalid log level", "LOG_LEVEL=info", "LOG_LEVEL=trace"},
		{"invalid log format", "LOG_FORMAT=text", "LOG_FORMAT=xml"},
		{"unsafe deployment profile", "APP_ENV=local", "APP_ENV=production"},
		{"retired sync mode", "LINE_EVENT_MODE=durable", "LINE_EVENT_MODE=sync"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !bytes.Contains(original, []byte(tt.before)) {
				t.Fatalf("example is missing expected safe setting %q", tt.before)
			}
			changed := strings.Replace(string(original), tt.before, tt.after, 1)
			if err := checkExample([]byte(changed)); err == nil {
				t.Fatalf("example with %q should fail runtime validation", tt.after)
			}
		})
	}
	// Even a known platform-provided key must not make a local template
	// accidentally pass as a Cloud Run configuration.
	if err := checkExample(append(append([]byte(nil), original...), []byte("K_SERVICE=unexpected-cloud-run\n")...)); err == nil {
		t.Fatal("local example must reject Cloud Run runtime metadata")
	}
}
