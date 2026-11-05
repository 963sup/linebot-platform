package architecture

import (
	"go/ast"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestDomainContextsAreIndependent prevents a convenient same-module import
// from becoming an undeclared Shared Kernel between bounded contexts.
func TestDomainContextsAreIndependent(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "domain")
	walkProductionGoFiles(t, root, func(path string, file *ast.File) {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Error(err)
			return
		}
		contextName := strings.Split(filepath.ToSlash(rel), "/")[0]
		for _, imp := range file.Imports {
			value, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			const prefix = modulePath + "/internal/domain/"
			if !strings.HasPrefix(value, prefix) {
				continue
			}
			imported := strings.Split(strings.TrimPrefix(value, prefix), "/")[0]
			if imported != contextName {
				t.Errorf("%s: bounded context %q imports aggregate internals from %q", path, contextName, imported)
			}
		}
	})
}

func TestApplicationHasNoProviderSDKTypes(t *testing.T) {
	walkProductionGoFiles(t, filepath.Join("..", "..", "internal", "application"), func(path string, file *ast.File) {
		for _, imp := range file.Imports {
			value, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			if isThirdParty(value) {
				t.Errorf("%s: Application imports provider or third-party package %q", path, value)
			}
		}
	})
}

func TestImplementedContextDirectoriesContainBehavior(t *testing.T) {
	for _, contextName := range []string{"identity", "messaging-delivery", "audience", "experience-presentation", "promotion", "workspace-automation"} {
		root := filepath.Join("..", "..", "internal", "domain", contextName)
		production, tests := 0, 0
		err := filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
				return nil
			}
			if strings.HasSuffix(info.Name(), "_test.go") {
				tests++
			} else {
				production++
			}
			return nil
		})
		if err != nil {
			t.Errorf("%s: %v", contextName, err)
			continue
		}
		if production == 0 || tests == 0 {
			t.Errorf("%s: requires production behavior and tests", contextName)
		}
	}
}
