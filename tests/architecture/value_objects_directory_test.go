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

// TestValueObjectsDirectoryContract checks tactical Domain model directories
// for the agreed value-objects/ directory and package name. Directory presence
// does not strategically promote a candidate Context.
func TestValueObjectsDirectoryContract(t *testing.T) {
	domain := filepath.Join("..", "..", "internal", "domain")
	identityValueDir := filepath.Join(domain, "identity", "value-objects")
	if info, err := os.Stat(identityValueDir); err != nil || !info.IsDir() {
		t.Fatalf("identity value objects must exist at %s: %v", identityValueDir, err)
	}
	err := filepath.WalkDir(domain, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		switch entry.Name() {
		case "valueobjects", "value_objects":
			return fmt.Errorf("forbidden value object directory %s: use value-objects", path)
		case "value-objects":
			return filepath.WalkDir(path, func(filePath string, file fs.DirEntry, fileErr error) error {
				if fileErr != nil {
					return fileErr
				}
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
					return nil
				}
				parsed, err := parser.ParseFile(token.NewFileSet(), filePath, nil, parser.PackageClauseOnly)
				if err != nil {
					return fmt.Errorf("parse package in %s: %w", filePath, err)
				}
				expected := "valueobjects"
				if strings.HasSuffix(file.Name(), "_test.go") && parsed.Name.Name == "valueobjects_test" {
					return nil
				}
				if parsed.Name.Name != expected {
					return fmt.Errorf("%s declares package %q, want %s", filePath, parsed.Name.Name, expected)
				}
				return nil
			})
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
