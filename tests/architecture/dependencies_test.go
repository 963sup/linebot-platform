package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	modulePath   = "github.com/963sup/linebot-platform"
	internalPath = modulePath + "/internal/"
)

func TestDependencyDirection(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		validate func(string) error
	}{
		{"domain only imports standard library and domain", "../../internal/domain", func(p string) error {
			if isThirdParty(p) {
				return fmt.Errorf("third-party dependency %q", p)
			}
			if strings.HasPrefix(p, internalPath) && !strings.HasPrefix(p, internalPath+"domain") {
				return fmt.Errorf("dependency outside Domain %q", p)
			}
			return nil
		}},
		{"application only imports standard library, application, and domain", "../../internal/application", func(p string) error {
			if isThirdParty(p) {
				return fmt.Errorf("third-party dependency %q", p)
			}
			if strings.HasPrefix(p, internalPath) && !strings.HasPrefix(p, internalPath+"application") && !strings.HasPrefix(p, internalPath+"domain") {
				return fmt.Errorf("dependency outside Application or Domain %q", p)
			}
			return nil
		}},
		{"inbound adapters do not import outbound adapters or infrastructure", "../../internal/adapter/inbound", func(p string) error {
			if strings.HasPrefix(p, internalPath+"adapter/outbound") || strings.HasPrefix(p, internalPath+"infrastructure") {
				return fmt.Errorf("forbidden inward dependency %q", p)
			}
			return nil
		}},
		{"outbound adapters do not import inbound adapters or bootstrap", "../../internal/adapter/outbound", func(p string) error {
			if strings.HasPrefix(p, internalPath+"adapter/inbound") || strings.HasPrefix(p, internalPath+"bootstrap") {
				return fmt.Errorf("forbidden outer-layer dependency %q", p)
			}
			return nil
		}},
		{"infrastructure does not import application, domain, or adapters", "../../internal/infrastructure", func(p string) error {
			for _, forbidden := range []string{"application", "domain", "adapter"} {
				if strings.HasPrefix(p, internalPath+forbidden) {
					return fmt.Errorf("forbidden outer-to-inner dependency %q", p)
				}
			}
			return nil
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			walkProductionGoFiles(t, tc.root, func(path string, file *ast.File) {
				for _, imp := range file.Imports {
					p, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						t.Errorf("%s: invalid import %s: %v", path, imp.Path.Value, err)
						continue
					}
					if err := tc.validate(p); err != nil {
						t.Errorf("%s: %v", path, err)
					}
				}
			})
		})
	}
}

// TestApplicationSerializationBoundary keeps transport serialization out of
// Application. The accepted baseline is intentionally empty.
func TestApplicationSerializationBoundary(t *testing.T) {
	want := map[string]bool{}
	got := make(map[string]bool)

	walkProductionGoFiles(t, "../../internal/application", func(path string, file *ast.File) {
		rel, err := filepath.Rel(filepath.Join("..", ".."), path)
		if err != nil {
			t.Errorf("relative path for %s: %v", path, err)
			return
		}
		rel = filepath.ToSlash(rel)
		for _, imp := range file.Imports {
			if imp.Path.Value == `"encoding/json"` {
				got[rel+":encoding/json"] = true
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok || field.Tag == nil {
				return true
			}
			tag, err := strconv.Unquote(field.Tag.Value)
			if err == nil && strings.Contains(tag, `json:`) {
				got[rel+":json-tag"] = true
			}
			return true
		})
	})

	assertExactBaseline(t, want, got)
}

func walkProductionGoFiles(t *testing.T, root string, visit func(string, *ast.File)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		visit(path, file)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func isThirdParty(importPath string) bool {
	if strings.HasPrefix(importPath, modulePath+"/") {
		return false
	}
	first, _, _ := strings.Cut(importPath, "/")
	return strings.Contains(first, ".")
}

func assertExactBaseline(t *testing.T, want, got map[string]bool) {
	t.Helper()
	for violation := range got {
		if !want[violation] {
			t.Errorf("new Application serialization violation: %s", violation)
		}
	}
	for violation := range want {
		if !got[violation] {
			t.Errorf("stale Application serialization baseline (remove it): %s", violation)
		}
	}
}
