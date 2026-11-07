// Package firestore creates native Google Cloud Firestore clients using
// application default credentials or explicitly supplied client options.
package firestore

import (
	"context"
	"errors"
	"strings"

	sdk "cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// NewClient returns the native default-database Firestore client. The caller
// owns Close(). Use a dedicated GCP project and least-privilege workload
// identity.
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*sdk.Client, error) {
	if ctx == nil {
		return nil, errors.New("Firestore context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("Firestore project ID is required")
	}
	return sdk.NewClient(ctx, projectID, opts...)
}
