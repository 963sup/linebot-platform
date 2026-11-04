package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	"golang.org/x/oauth2"
)

// workspacePrivateFile rejects non-regular/symlink/shared-permission files.
// Never include file contents or bearer tokens in error messages.
func workspacePrivateFile(path string, limit int64) ([]byte, error) {
	if path == "" {
		return nil, errors.New("private workspace file required")
	}
	info, err := os.Lstat(filepath.Clean(path))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("workspace input must be a regular owner-only file")
	}
	return readPrivateBounded(path, limit)
}

// runWorkspace is intentionally local-only and never exposed on a Cloud Run
// endpoint. It requires an externally issued, short-lived user OAuth access
// token; refresh tokens are not accepted. A proper OAuth consent flow and
// secure persistent token repository are separate infrastructure stages.
func runWorkspace(args []string, getenv func(string) string) error {
	flags := flag.NewFlagSet("workspace", flag.ContinueOnError)
	api := flags.String("api", "", "calendar|tasks|forms")
	requestFile := flags.String("request", "", "0600 JSON file with operation/parameters/options/body")
	tokenFile := flags.String("token-file", "", "0600 file containing a short-lived OAuth access token")
	output := flags.String("out", "", "new private result JSON file")
	read := flags.Bool("read", false, "authorize read-only operation")
	apply := flags.Bool("apply", false, "authorize remote mutation")
	confirm := flags.String("confirm", "", "must match operation for mutation")
	highRisk := flags.String("confirm-dangerous", "", "additional confirmation for destructive operation")
	testAccount := flags.Bool("test-account", false, "acknowledge isolated test account")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || getenv == nil || !*testAccount || *requestFile == "" || *tokenFile == "" || *output == "" {
		return errors.New("workspace requires --test-account --request --token-file --out")
	}
	if getenv("K_SERVICE") != "" || getenv("APP_ENV") != "local" {
		return errors.New("workspace operator requires explicit local environment")
	}
	if *api != "calendar" && *api != "tasks" && *api != "forms" {
		return errors.New("workspace API must be calendar, tasks or forms")
	}
	data, err := workspacePrivateFile(*requestFile, 1<<20)
	if err != nil {
		return err
	}
	var req outbound.WorkspaceRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return errors.New("invalid workspace request JSON")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("workspace request contains trailing data")
	}
	if strings.TrimSpace(req.Operation) == "" {
		return errors.New("workspace operation is required")
	}
	approval := workspace.Approval{Read: *read, Apply: *apply, Confirm: *confirm, ConfirmDangerous: *highRisk}
	if err := workspace.Validate(*api, req, approval); err != nil {
		return err
	}
	tokenBytes, err := workspacePrivateFile(*tokenFile, 16<<10)
	if err != nil {
		return err
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" || strings.ContainsAny(token, " \t\n\r") {
		return errors.New("invalid OAuth access token file")
	}
	return writePrivateOperation(*output, func() ([]byte, error) {
		if workspaceRuntimeFactory == nil {
			return nil, errors.New("workspace runtime is not configured")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token, TokenType: "Bearer"})
		ops, err := workspaceRuntimeFactory(ctx, source)
		if err != nil {
			return nil, err
		}
		result, err := ops.Execute(ctx, *api, req, approval)
		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			return []byte("{}"), nil
		}
		return result, nil
	})
}
