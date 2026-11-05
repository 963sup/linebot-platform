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

const sharedGoogleAPICodec = "internal/adapter/outbound/google/api/codec"

func TestOutboundAdapterPackagesDeclareOwnedContracts(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "adapter", "outbound")
	packages := make(map[string][]*ast.File)
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
		packages[filepath.Dir(path)] = append(packages[filepath.Dir(path)], file)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for dir, files := range packages {
		rel, err := filepath.Rel(filepath.Join("..", ".."), dir)
		if err != nil {
			t.Error(err)
			continue
		}
		if filepath.ToSlash(rel) == sharedGoogleAPICodec {
			continue
		}
		if !hasOwnedContractAssertion(files) {
			t.Errorf("%s: outbound adapter package must declare a compile-time assertion for an Application capability port or Domain repository contract", filepath.ToSlash(rel))
		}
	}
}

func TestOwnedContractAssertionClassification(t *testing.T) {
	tests := map[string]struct {
		source string
		want   bool
	}{
		"application capability": {`package p; import app "github.com/963sup/linebot-platform/internal/application/usecases/example"; var _ app.Port = (*Adapter)(nil)`, true},
		"domain repository":      {`package p; import repo "github.com/963sup/linebot-platform/internal/domain/example/repositories"; var _ repo.Store = (*Adapter)(nil)`, true},
		"infrastructure":         {`package p; import infra "github.com/963sup/linebot-platform/internal/infrastructure/example"; var _ infra.Client = (*Adapter)(nil)`, false},
		"peer adapter":           {`package p; import peer "github.com/963sup/linebot-platform/internal/adapter/outbound/peer"; var _ peer.Port = (*Adapter)(nil)`, false},
		"no assertion":           {`package p; type Adapter struct{}`, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), name+".go", test.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := hasOwnedContractAssertion([]*ast.File{file}); got != test.want {
				t.Fatalf("hasOwnedContractAssertion() = %v, want %v", got, test.want)
			}
		})
	}
}

func hasOwnedContractAssertion(files []*ast.File) bool {
	for _, file := range files {
		imports := make(map[string]string)
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			name := filepath.Base(path)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			imports[name] = path
		}
		for _, declaration := range file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || len(value.Names) != 1 || value.Names[0].Name != "_" {
					continue
				}
				selector, ok := value.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				pkg, ok := selector.X.(*ast.Ident)
				if !ok || !isOwnedContractPackage(imports[pkg.Name]) {
					continue
				}
				return true
			}
		}
	}
	return false
}

func isOwnedContractPackage(path string) bool {
	if isPackageWithin(path, internalPath+"application/usecases") {
		return true
	}
	const domainPrefix = internalPath + "domain/"
	if !strings.HasPrefix(path, domainPrefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, domainPrefix), "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] == "repositories"
}

func TestSharedGoogleAPICodecRemainsPureMapping(t *testing.T) {
	root := filepath.Join("..", "..", filepath.FromSlash(sharedGoogleAPICodec))
	violations := []string{}
	walkProductionGoFiles(t, root, func(path string, file *ast.File) {
		violations = append(violations, codecPurityViolations(file)...)
	})
	for _, violation := range violations {
		t.Error(violation)
	}
}

func TestCodecPurityClassification(t *testing.T) {
	tests := map[string]struct {
		source string
		clean  bool
	}{
		"pure mapper":         {`package p; import "encoding/json"; func Map(v any) ([]byte, error) { return json.Marshal(v) }`, true},
		"adapter import":      {`package p; import _ "github.com/963sup/linebot-platform/internal/adapter/outbound/peer"`, false},
		"infrastructure":      {`package p; import _ "github.com/963sup/linebot-platform/internal/infrastructure/httpclient"`, false},
		"network client":      {`package p; import _ "net/http"`, false},
		"goroutine":           {`package p; func Map() { go func(){}() }`, false},
		"package state":       {`package p; var cache = map[string]string{}`, false},
		"initialization hook": {`package p; func init() {}`, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), name+".go", test.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(codecPurityViolations(file)) == 0; got != test.clean {
				t.Fatalf("codec clean = %v, want %v", got, test.clean)
			}
		})
	}
}

func codecPurityViolations(file *ast.File) []string {
	violations := []string{}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			violations = append(violations, "invalid codec import")
			continue
		}
		if isPackageWithin(path, internalPath+"adapter") || isPackageWithin(path, internalPath+"infrastructure") ||
			isPackageWithin(path, internalPath+"bootstrap") || path == "net" || strings.HasPrefix(path, "net/") ||
			path == "os" || strings.HasPrefix(path, "os/") {
			violations = append(violations, "codec imports effect or peer package "+path)
		}
	}
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.GenDecl:
			if value.Tok == token.VAR {
				violations = append(violations, "codec declares package mutable state")
			}
		case *ast.FuncDecl:
			if value.Recv == nil && value.Name.Name == "init" {
				violations = append(violations, "codec declares init hook")
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if _, ok := node.(*ast.GoStmt); ok {
			violations = append(violations, "codec starts goroutine")
		}
		return true
	})
	return violations
}
