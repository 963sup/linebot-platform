package architecture

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var legacyUsecasePackages = []string{
	"channel",
	"contacts",
	"conversation",
	"engagement",
	"insights",
	"messaging",
	"workspace",
}

// TestFinalCutoverHasNoGlobalApplicationPorts is the final-cutover gate. A
// port belongs to the capability that consumes it; a global bucket obscures
// ownership and encourages provider-shaped interfaces.
func TestFinalCutoverHasNoGlobalApplicationPorts(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "application", "ports")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("final cutover incomplete: remove %s after every consumer imports its capability-local contract", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// TestFinalCutoverHasNoLegacyUsecases rejects both old package names and
// imports of them. Keeping aliases would preserve two owners for one behavior.
func TestFinalCutoverHasNoLegacyUsecases(t *testing.T) {
	legacy := make(map[string]bool, len(legacyUsecasePackages))
	var violations []string
	for _, name := range legacyUsecasePackages {
		legacy[name] = true
		path := filepath.Join("..", "..", "internal", "application", "usecases", name)
		if _, err := os.Stat(path); err == nil {
			violations = append(violations, "legacy package remains: "+filepath.ToSlash(path))
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}

	walkProductionGoFiles(t, filepath.Join("..", ".."), func(path string, file *ast.File) {
		for _, imp := range file.Imports {
			value, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			const prefix = modulePath + "/internal/application/usecases/"
			if !strings.HasPrefix(value, prefix) {
				continue
			}
			name := strings.Split(strings.TrimPrefix(value, prefix), "/")[0]
			if legacy[name] {
				rel, relErr := filepath.Rel(filepath.Join("..", ".."), path)
				if relErr != nil {
					t.Fatal(relErr)
				}
				violations = append(violations, fmt.Sprintf("%s imports legacy usecase %q", filepath.ToSlash(rel), name))
			}
		}
	})
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("final cutover incomplete:\n%s", strings.Join(violations, "\n"))
	}
}

// TestApplicationPortsAreConsumerOwned prevents one capability from importing
// another capability's interfaces. Cross-context collaboration must use a
// consumer-owned contract, DTO, or event at the consuming boundary.
func TestApplicationPortsAreConsumerOwned(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "application", "usecases")
	walkProductionGoFiles(t, root, func(path string, file *ast.File) {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Error(err)
			return
		}
		owner := strings.Split(filepath.ToSlash(rel), "/")[0]
		for _, imp := range file.Imports {
			value, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			const prefix = modulePath + "/internal/application/usecases/"
			if !strings.HasPrefix(value, prefix) {
				continue
			}
			provider := strings.Split(strings.TrimPrefix(value, prefix), "/")[0]
			if provider != owner {
				t.Errorf("%s: capability %q imports sibling capability %q; define the contract at the consuming boundary", path, owner, provider)
			}
		}
	})
}
