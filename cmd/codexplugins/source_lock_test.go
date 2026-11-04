package main

import (
	"path/filepath"
	"testing"
)

func TestRepositorySourceLock(t *testing.T) {
	sources, err := loadSources(filepath.Join("..", "..", ".codex", "plugins.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != len(approvedSources) {
		t.Fatal("source lock must cover both approved sources exactly once")
	}
}
