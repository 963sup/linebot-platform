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
		"LINE_CHANNEL_ACCESS_TOKEN=private-token\n",
		"LINE_CHANNEL_SECRET=private-secret\n",
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
