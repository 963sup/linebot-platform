package architecture

import (
	"go/ast"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
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

func TestTacticalDomainModelDirectoriesContainBehavior(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "docs", "domain", "model.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Contexts []struct {
			ID      string `yaml:"id"`
			Package string `yaml:"package"`
		} `yaml:"contexts"`
	}
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("decode %s: %v", manifestPath, err)
	}

	registered := make(map[string]string)
	for _, context := range manifest.Contexts {
		const prefix = "internal/domain/"
		if !strings.HasPrefix(context.Package, prefix) {
			continue
		}
		contextName := strings.TrimPrefix(context.Package, prefix)
		if contextName == "" || strings.Contains(contextName, "/") {
			t.Errorf("%s: context %q has invalid tactical Domain package %q", manifestPath, context.ID, context.Package)
			continue
		}
		if previous, exists := registered[contextName]; exists {
			t.Errorf("domain directory %q is registered by both contexts %q and %q", contextName, previous, context.ID)
			continue
		}
		registered[contextName] = context.ID
		root := filepath.Join("..", "..", "internal", "domain", contextName)
		production, tests, err := countGoBehavior(root)
		if err != nil {
			t.Errorf("%s: %v", contextName, err)
			continue
		}
		if production == 0 || tests == 0 {
			t.Errorf("%s: requires production behavior and tests", contextName)
		}
	}
	if len(registered) == 0 {
		t.Fatal("domain manifest defines no tactical Domain model packages")
	}

	domainRoot := filepath.Join("..", "..", "internal", "domain")
	entries, err := os.ReadDir(domainRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		production, _, err := countGoBehavior(filepath.Join(domainRoot, entry.Name()))
		if err != nil {
			t.Errorf("%s: %v", entry.Name(), err)
			continue
		}
		if production > 0 {
			if _, exists := registered[entry.Name()]; !exists {
				t.Errorf("domain directory %q contains production Go but is not uniquely registered by a context package", entry.Name())
			}
		}
	}
}

func countGoBehavior(root string) (production, tests int, err error) {
	err = filepath.WalkDir(root, func(path string, info fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
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
	return production, tests, err
}
