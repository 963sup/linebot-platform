package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	"golang.org/x/oauth2"
)

type blockingWorkspaceRuntime struct{ entered chan context.Context }

func (r *blockingWorkspaceRuntime) Execute(ctx context.Context, _ workspaceautomation.API, _ workspaceautomation.OperationCommand, _ workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error) {
	r.entered <- ctx
	<-ctx.Done()
	return workspaceautomation.OperationResult{}, ctx.Err()
}

func TestWorkspaceOpsMustFailClosed(t *testing.T) {
	env := func(key string) string {
		if key == "APP_ENV" {
			return "local"
		}
		return ""
	}
	for _, args := range [][]string{
		{"--api", "calendar", "--read"},
		{"--api", "calendar", "--test-account", "--read", "--request", "nonexistent", "--token-file", "missing", "--out", "out"},
		{"--api", "tasks", "--test-account", "--request", "nonexistent", "--token-file", "missing", "--out", "out", "--apply", "--confirm", "tasks.delete"},
	} {
		if err := runWorkspace(testContext(), args, env); err == nil {
			t.Fatalf("invalid workspace CLI accepted: %v", args)
		}
	}
	dir := t.TempDir()
	req := filepath.Join(dir, "request.json")
	token := filepath.Join(dir, "access_token.txt")
	if err := os.WriteFile(req, []byte(`{"operation":"events.insert","parameters":{"calendarId":"primary"},"body":{"summary":"Test"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(token, []byte("fake-test-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--api", "calendar", "--test-account", "--request", req, "--token-file", token, "--out", filepath.Join(dir, "output.json"), "--apply"}
	if err := runWorkspace(testContext(), args, env); err == nil {
		t.Fatal("write without matching confirmation accepted")
	}
	if err := runWorkspace(testContext(), append(args, "--confirm", "events.delete"), env); err == nil {
		t.Fatal("wrong write confirmation accepted")
	}
	prod := func(key string) string {
		if key == "APP_ENV" {
			return "production"
		}
		return ""
	}
	if err := runWorkspace(testContext(), append(args, "--confirm", "events.insert"), prod); err == nil {
		t.Fatal("production environment accepted")
	}
	if err := os.Chmod(token, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := workspacePrivateFile(token, 1024); err == nil {
		t.Fatal("shared token file accepted")
	}
}

func TestWorkspacePropagatesParentContextDuringExecution(t *testing.T) {
	dir := t.TempDir()
	request := filepath.Join(dir, "request.json")
	token := filepath.Join(dir, "token.txt")
	output := filepath.Join(dir, "output.json")
	if err := os.WriteFile(request, []byte(`{"operation":"events.list","parameters":{"calendarId":"primary"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(token, []byte("test-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	type contextKey struct{}
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "workspace"))
	runtime := &blockingWorkspaceRuntime{entered: make(chan context.Context, 1)}
	deps := Dependencies{WorkspaceRuntime: func(ctx context.Context, _ oauth2.TokenSource) (WorkspaceRuntime, error) {
		if ctx.Value(contextKey{}) != "workspace" {
			t.Fatal("workspace factory context value was not propagated")
		}
		return runtime, nil
	}}
	done := make(chan error, 1)
	go func() {
		done <- runWorkspace(parent, []string{"--api", "calendar", "--test-account", "--request", request, "--token-file", token, "--out", output, "--read"}, func(key string) string {
			if key == "APP_ENV" {
				return "local"
			}
			return ""
		}, deps)
	}()
	ctx := <-runtime.entered
	deadline, ok := ctx.Deadline()
	if ctx.Value(contextKey{}) != "workspace" || !ok || time.Until(deadline) > 15*time.Second {
		t.Fatalf("workspace context value/deadline not propagated: %v %v", ctx.Value(contextKey{}), deadline)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("workspace cancellation error = %v", err)
	}
}
