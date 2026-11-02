//go:build dependencies

// Package dependencies tracks optional SDK modules without activating them in runtime.
// This build-tag-only file allows go mod tidy to retain preapproved SDK dependencies.
package dependencies

import (
	_ "cloud.google.com/go/cloudtasks/apiv2"
	_ "cloud.google.com/go/firestore"
	_ "cloud.google.com/go/secretmanager/apiv1"
	_ "cloud.google.com/go/storage"
	_ "github.com/openai/openai-go/v3"
	_ "go.linecorp.com/garr/retry"
	_ "google.golang.org/api/calendar/v3"
	_ "google.golang.org/api/drive/v3"
	_ "google.golang.org/api/sheets/v4"
	_ "google.golang.org/genai"
)
