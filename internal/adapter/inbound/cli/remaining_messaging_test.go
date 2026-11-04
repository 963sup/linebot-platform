package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

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
		if err := runActions(args, env); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if err := runLinkToken([]string{"--user", "U1", "--out", "x", "--apply"}, env); err == nil {
		t.Fatal("link token issued without confirmation")
	}
	var out bytes.Buffer
	if err := runEligibility([]string{"--action", "push-phone", "--apply", "--confirm", "push-phone"}, env, &out); err == nil {
		t.Fatal("PNP without review")
	}
	if err := runEligibility([]string{"--action", "pnp-stats", "--id", "20261101", "--read"}, env, &out); err == nil {
		t.Fatal("stats without review")
	}
	if err := runMenuImage([]string{"--menu-id", "richmenu-1", "--out", "x"}, env); err == nil {
		t.Fatal("menu image without read")
	}
	if out.Len() != 0 {
		t.Fatal("rejected action printed data")
	}
}
