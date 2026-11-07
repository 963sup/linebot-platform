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

const DefaultDatabase = "(default)"

// NewClient returns the native default-database Firestore client. The caller
// owns Close(). Use a dedicated GCP project and least-privilege workload
// identity.
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*sdk.Client, error) {
	return NewClientWithDatabase(ctx, projectID, DefaultDatabase, opts...)
}

// NewClientWithDatabase returns a native client for an explicit Firestore
// database. The caller owns Close().
func NewClientWithDatabase(ctx context.Context, projectID, databaseID string, opts ...option.ClientOption) (*sdk.Client, error) {
	if ctx == nil {
		return nil, errors.New("Firestore context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("Firestore project ID is required")
	}
	if err := ValidateDatabaseID(databaseID); err != nil {
		return nil, err
	}
	return sdk.NewClientWithDatabase(ctx, projectID, databaseID, opts...)
}

func ValidateDatabaseID(databaseID string) error {
	if strings.TrimSpace(databaseID) == "" || databaseID == "." || databaseID == ".." || databaseID != strings.TrimSpace(databaseID) || strings.ContainsAny(databaseID, "/\\\x00\r\n\t ") {
		return errors.New("Firestore database ID is required")
	}
	return nil
}
