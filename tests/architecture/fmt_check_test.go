package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFmtCheckFailsWhenGofmtIsUnavailable(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make is unavailable")
	}
	command := exec.Command(makePath, "fmt-check")
	command.Dir = filepath.Join("..", "..")
	command.Env = append(os.Environ(), "PATH="+t.TempDir())
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("fmt-check passed without gofmt")
	}
	if !strings.Contains(string(output), "gofmt is required for fmt-check") {
		t.Fatalf("fmt-check failed for the wrong reason: %s", output)
	}
}

func TestFmtCheckFailsWhenGofmtExecutionFails(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make is unavailable")
	}
	bin := t.TempDir()
	gofmt := filepath.Join(bin, "gofmt")
	if err := os.WriteFile(gofmt, []byte("#!/bin/sh\nexit 42\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(makePath, "fmt-check")
	command.Dir = filepath.Join("..", "..")
	command.Env = append(os.Environ(), "PATH="+bin)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("fmt-check passed after gofmt execution failed")
	}
	if !strings.Contains(string(output), "gofmt failed during fmt-check") {
		t.Fatalf("fmt-check failed for the wrong reason: %s", output)
	}
}
