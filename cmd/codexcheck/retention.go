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
	if info, err := os.Stat(filepath.Join(root, "internal", "infrastructure", "dependencies", "modules.go")); err != nil || info.IsDir() {
		return fmt.Errorf("preapproved optional SDK module tracking file must be retained")
	}
	return nil
}
