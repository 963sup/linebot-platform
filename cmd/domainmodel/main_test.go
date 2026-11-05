package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunChecksRepositoryProjection(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := run("check"); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsUnknownMode(t *testing.T) {
	if err := run("invalid"); err == nil {
		t.Fatal("expected unknown mode to fail")
	}
}
