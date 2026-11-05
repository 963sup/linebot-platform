package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCoreLayersDoNotStartGoroutines(t *testing.T) {
	for _, root := range []string{"../../internal/domain", "../../internal/application"} {
		walkProductionGoFiles(t, root, func(path string, file *ast.File) {
			ast.Inspect(file, func(node ast.Node) bool {
				if _, ok := node.(*ast.GoStmt); ok {
					t.Errorf("%s: Domain and Application must not own goroutine lifecycle; move bounded concurrency to an adapter or infrastructure boundary", path)
				}
				return true
			})
		})
	}
}

func TestCoreGoroutineGuardDetectsGoStatements(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "synthetic.go", "package synthetic\nfunc f() { go f() }", 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if found {
			return false
		}
		if _, ok := node.(*ast.GoStmt); ok {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Fatal("GoStmt guard failed to detect a goroutine")
	}
}

func TestProductionCLIRequiresInjectedContext(t *testing.T) {
	walkProductionGoFiles(t, "../../internal/adapter/inbound/cli", func(path string, file *ast.File) {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 0 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (selector.Sel.Name != "Background" && selector.Sel.Name != "TODO") {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "context" {
				t.Errorf("%s: production CLI must derive work from its injected root context, not context.%s()", path, selector.Sel.Name)
			}
			return true
		})
	})
}

func TestCLIContextGuardDetectsDetachedRoots(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "synthetic.go", "package synthetic\nimport \"context\"\nfunc f() { _, _ = context.Background(), context.TODO() }", 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if ok && pkg.Name == "context" && (selector.Sel.Name == "Background" || selector.Sel.Name == "TODO") {
			found++
		}
		return true
	})
	if found != 2 {
		t.Fatalf("context guard found %d detached roots, want 2", found)
	}
}
