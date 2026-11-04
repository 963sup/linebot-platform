// Package storage creates native Google Cloud Storage clients.
package storage

import (
	"context"
	"errors"

	sdk "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// NewClient uses ADC/workload identity or provided options. Caller owns Close.
// It performs no bucket or object writes during construction.
func NewClient(ctx context.Context, opts ...option.ClientOption) (*sdk.Client, error) {
	if ctx == nil {
		return nil, errors.New("Cloud Storage context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return sdk.NewClient(ctx, opts...)
}
