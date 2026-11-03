// Package cloudtasks constructs the native Google Cloud Tasks client.
// This is a durable background task queue, not the Google Tasks to-do API.
package cloudtasks

import (
	"context"
	"errors"

	sdk "cloud.google.com/go/cloudtasks/apiv2"
	"google.golang.org/api/option"
)

// NewClient uses ADC/workload identity unless explicit options override it.
// Caller owns Close(). Nothing is enqueued by constructing the client.
func NewClient(ctx context.Context, opts ...option.ClientOption) (*sdk.Client, error) {
	if ctx == nil {
		return nil, errors.New("Cloud Tasks context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return sdk.NewClient(ctx, opts...)
}
