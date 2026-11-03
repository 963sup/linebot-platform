// Package api constructs native Google Workspace clients for an explicitly
// authorized user identity. It does not perform OAuth consent or store tokens.
package api

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/tasks/v1"
)

// validateUserClient deliberately requires caller-owned authenticated HTTP.
// Do not pass a bare Cloud Run/ADC transport and assume it can access a
// person's Tasks or Calendar. The caller must obtain a user-scoped OAuth
// client with the required scopes and manage its token lifecycle securely.
func validateUserClient(ctx context.Context, authorizedHTTP *http.Client) error {
	if ctx == nil {
		return errors.New("workspace context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if authorizedHTTP == nil {
		return errors.New("user-authorized workspace HTTP client is required")
	}
	return nil
}

func NewTasks(ctx context.Context, authorizedHTTP *http.Client) (*tasks.Service, error) {
	if err := validateUserClient(ctx, authorizedHTTP); err != nil {
		return nil, err
	}
	return tasks.NewService(ctx, option.WithHTTPClient(authorizedHTTP))
}

func NewCalendar(ctx context.Context, authorizedHTTP *http.Client) (*calendar.Service, error) {
	if err := validateUserClient(ctx, authorizedHTTP); err != nil {
		return nil, err
	}
	return calendar.NewService(ctx, option.WithHTTPClient(authorizedHTTP))
}

func NewSheets(ctx context.Context, authorizedHTTP *http.Client) (*sheets.Service, error) {
	if err := validateUserClient(ctx, authorizedHTTP); err != nil {
		return nil, err
	}
	return sheets.NewService(ctx, option.WithHTTPClient(authorizedHTTP))
}

func NewDrive(ctx context.Context, authorizedHTTP *http.Client) (*drive.Service, error) {
	if err := validateUserClient(ctx, authorizedHTTP); err != nil {
		return nil, err
	}
	return drive.NewService(ctx, option.WithHTTPClient(authorizedHTTP))
}
