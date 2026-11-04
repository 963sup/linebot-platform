package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// protectedSDKModules includes preapproved, potentially unused dependencies.
// Release cleanup must preserve their declared go.mod versions and go.sum.
var protectedSDKModules = []string{
	"cloud.google.com/go/cloudtasks",
	"cloud.google.com/go/firestore",
	"cloud.google.com/go/secretmanager",
	"cloud.google.com/go/storage",
	"github.com/BurntSushi/toml",
	"github.com/google/uuid",
	"github.com/line/line-bot-sdk-go/v8",
	"github.com/openai/openai-go/v3",
	"go.linecorp.com/garr",
	"golang.org/x/oauth2",
	"google.golang.org/api",
	"google.golang.org/genai",
	"google.golang.org/grpc",
}

// protectedSDKScaffolds are intentional integration boundaries. They may
// contain only .gitkeep today and must remain even while runtime-unused.
var protectedSDKScaffolds = []string{
	"internal/adapter/outbound/genai",
	"internal/adapter/outbound/google/api/calendar",
	"internal/adapter/outbound/google/api/drive",
	"internal/adapter/outbound/google/api/sheets",
	"internal/adapter/outbound/google/cloud/storage",
	"internal/adapter/outbound/openai",
	"internal/infrastructure/google/api/calendar",
	"internal/infrastructure/google/api/drive",
	"internal/infrastructure/google/api/oauth",
	"internal/infrastructure/google/api/sheets",
	"internal/infrastructure/google/cloud/secretmanager",
	"internal/infrastructure/google/cloud/storage",
	"internal/infrastructure/google/genai",
	"internal/infrastructure/openai",
}

func validateSDKRetention(root string) error {
	direct, err := directModules(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	for _, module := range protectedSDKModules {
		if _, ok := direct[module]; !ok {
			return fmt.Errorf("reserved SDK module must remain a direct go.mod dependency: %s", module)
		}
	}
	for _, directory := range protectedSDKScaffolds {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(directory)))
		if err != nil || !info.IsDir() {
			return fmt.Errorf("reserved SDK scaffold directory must be retained: %s", directory)
		}
	}
	if info, err := os.Stat(filepath.Join(root, "internal", "infrastructure", "dependencies", "modules.go")); err != nil || info.IsDir() {
		return fmt.Errorf("preapproved optional SDK module tracking file must be retained")
	}
	return nil
}
