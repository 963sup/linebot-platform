package contract

import (
	"context"
	"errors"
	"testing"

	cloudtasks "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/cloudtasks"
	firestore "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
	secretmanager "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/secretmanager"
)

func TestGoogleCloudFactoriesRejectInvalidContext(t *testing.T) {
	var nilContext context.Context
	if _, err := firestore.NewClient(nilContext, "test-project"); err == nil {
		t.Fatal("Firestore must reject nil context")
	}
	if _, err := firestore.NewClient(context.Background(), ""); err == nil {
		t.Fatal("Firestore must reject missing project")
	}
	if _, err := cloudtasks.NewClient(nilContext); err == nil {
		t.Fatal("Cloud Tasks must reject nil context")
	}
	if _, err := secretmanager.NewClient(nilContext); err == nil {
		t.Fatal("Secret Manager must reject nil context")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, construct := range map[string]func() error{
		"firestore": func() error {
			_, err := firestore.NewClient(ctx, "test-project")
			return err
		},
		"cloudtasks": func() error {
			_, err := cloudtasks.NewClient(ctx)
			return err
		},
		"secretmanager": func() error {
			_, err := secretmanager.NewClient(ctx)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := construct(); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation from %s factory, got %v", name, err)
			}
		})
	}
}
