package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	conversationoperations "github.com/963sup/linebot-platform/internal/application/usecases/conversation-operations"
)

type failingActionsRuntime struct {
	operationErr error
	closeErr     error
}

func (r failingActionsRuntime) Leave(context.Context, conversationoperations.Conversation) error {
	return r.operationErr
}
func (r failingActionsRuntime) ShowLoading(context.Context, conversationoperations.LoadingIndicator) error {
	return r.operationErr
}
func (r failingActionsRuntime) MarkRead(context.Context, conversationoperations.ReadTarget) error {
	return r.operationErr
}
func (r failingActionsRuntime) MarkReadByToken(context.Context, conversationoperations.ReadToken) error {
	return r.operationErr
}
func (r failingActionsRuntime) Close() error { return r.closeErr }

func TestPrivateFileIsExclusiveAnd0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	if err := writePrivateFile(path, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	mode, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode.Mode().Perm() != 0o600 {
		t.Fatalf("file permission %v", mode.Mode().Perm())
	}
	if err := writePrivateFile(path, []byte("secret")); err == nil {
		t.Fatal("overwrote existing file")
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "{}" {
		t.Fatal("existing data modified")
	}
	if _, err := readPrivateBounded(path, 1); err == nil {
		t.Fatal("over-limit file accepted")
	}
}

func TestMissingAuthorityCannotReachLINE(t *testing.T) {
	env := func(string) string { return "" }
	actions := [][]string{
		{"--action", "leave-room", "--id", "room1", "--apply"},
		{"--action", "leave-room", "--id", "room1", "--apply", "--confirm", "leave-group"},
		{"--action", "mark-read", "--apply", "--confirm", "mark-read"},
		{"--action", "loading", "--apply", "--confirm", "loading", "--request", "not-present.json"},
	}
	for _, args := range actions {
		if err := runActions(testContext(), args, env); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if err := runLinkToken(testContext(), []string{"--user", "U1", "--out", "x", "--apply"}, env); err == nil {
		t.Fatal("link token issued without confirmation")
	}
	var out bytes.Buffer
	if err := runEligibility(testContext(), []string{"--action", "push-phone", "--apply", "--confirm", "push-phone"}, env, &out); err == nil {
		t.Fatal("PNP without review")
	}
	if err := runEligibility(testContext(), []string{"--action", "pnp-stats", "--id", "20261101", "--read"}, env, &out); err == nil {
		t.Fatal("stats without review")
	}
	if err := runMenuImage(testContext(), []string{"--menu-id", "richmenu-1", "--out", "x"}, env); err == nil {
		t.Fatal("menu image without read")
	}
	if out.Len() != 0 {
		t.Fatal("rejected action printed data")
	}
}

func TestActionsJoinsOperationAndRuntimeCloseErrors(t *testing.T) {
	operationErr := errors.New("operation failed")
	closeErr := errors.New("runtime close failed")
	deps := Dependencies{ActionsRuntime: func(string) (ActionsRuntime, error) {
		return failingActionsRuntime{operationErr: operationErr, closeErr: closeErr}, nil
	}}
	err := runActions(testContext(), []string{
		"--action", "leave-group", "--id", "group-1", "--apply", "--confirm", "leave-group",
	}, func(string) string { return "token" }, deps)
	if !errors.Is(err, operationErr) || !errors.Is(err, closeErr) {
		t.Fatalf("runActions error=%v, want operation and close errors", err)
	}
}
