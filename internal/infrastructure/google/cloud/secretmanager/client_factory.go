// Package secretmanager creates the native Google Cloud Secret Manager client.
package secretmanager

import (
	"context"
	"errors"

	sdk "cloud.google.com/go/secretmanager/apiv1"
	"google.golang.org/api/option"
)

// NewClient returns an SDK client with caller-owned Close().
// Read only explicitly allowed secrets under least-privilege IAM; never log
// secret values or use this constructor as a network health check.
func NewClient(ctx context.Context, opts ...option.ClientOption) (*sdk.Client, error) {
	if ctx == nil {
		return nil, errors.New("Secret Manager context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return sdk.NewClient(ctx, opts...)
}
