package architecture

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValueObjectsDirectoryContract prevents tooling and future agents
// from silently reverting the agreed domain directory naming.
func TestValueObjectsDirectoryContract(t *testing.T) {
	domain := filepath.Join("..", "..", "internal", "domain")
	valueDir := filepath.Join(domain, "value-objects")
	info, err := os.Stat(valueDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("value object directory must exist at %s: %v", valueDir, err)
	}
	legacyDir := filepath.Join(domain, "valueobjects")
	if _, err := os.Stat(legacyDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy directory must not exist at %s: %v", legacyDir, err)
	}
	err = filepath.WalkDir(valueDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
		if err != nil {
			return fmt.Errorf("parse package in %s: %w", path, err)
		}
		if file.Name.Name == "valueobjects" ||
			(strings.HasSuffix(entry.Name(), "_test.go") && file.Name.Name == "valueobjects_test") {
			return nil
		}
		return fmt.Errorf("%s declares package %q, want valueobjects", path, file.Name.Name)
	})
	if err != nil {
		t.Fatal(err)
	}
}
