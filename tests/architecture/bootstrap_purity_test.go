package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapDoesNotOwnInboundPresentationMapping(t *testing.T) {
	contracts := cliRuntimeContracts(t)
	receiverMethods := make(map[string]map[string]bool)
	receiverPaths := make(map[string]string)

	walkProductionGoFiles(t, "../../internal/bootstrap", func(path string, file *ast.File) {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			receiver := receiverName(fn.Recv.List[0].Type)
			if receiver == "" {
				continue
			}
			if receiverMethods[receiver] == nil {
				receiverMethods[receiver] = make(map[string]bool)
			}
			receiverMethods[receiver][fn.Name.Name] = true
			receiverPaths[receiver] = path
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			selector, ok := literal.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "cli" && selector.Sel.Name != "Dependencies" && selector.Sel.Name != "RuntimeAdapters" {
				t.Errorf("%s: bootstrap constructs inbound presentation type cli.%s", path, selector.Sel.Name)
			}
			return true
		})
	})

	for receiver, methods := range receiverMethods {
		for contract, required := range contracts {
			if implementsSemanticMethods(methods, required) {
				t.Errorf("%s: bootstrap type %s implements inbound CLI contract %s; move presentation adaptation to internal/adapter/inbound/cli", receiverPaths[receiver], receiver, contract)
			}
		}
	}
}

func TestBootstrapRuntimeContractDetectionIsNameIndependent(t *testing.T) {
	contract := cliRuntimeContracts(t)["WorkspaceRuntime"]
	if !implementsSemanticMethods(map[string]bool{"Execute": true}, contract) {
		t.Fatal("a bootstrap type named arbitrarily must still be detected when it implements WorkspaceRuntime")
	}
}

// cliRuntimeContracts derives ownership from the runtime factories exposed by
// cli.Dependencies. It intentionally does not rely on type-name suffixes.
func cliRuntimeContracts(t *testing.T) map[string]map[string]bool {
	t.Helper()
	files, err := parser.ParseDir(token.NewFileSet(), "../../internal/adapter/inbound/cli", func(info fs.FileInfo) bool {
		return filepath.Ext(info.Name()) == ".go" && !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}

	interfaces := make(map[string]*ast.InterfaceType)
	contractNames := make(map[string]bool)
	for _, pkg := range files {
		for _, file := range pkg.Files {
			for _, declaration := range file.Decls {
				gen, ok := declaration.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if iface, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						interfaces[typeSpec.Name.Name] = iface
					}
					if typeSpec.Name.Name != "Dependencies" {
						continue
					}
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						t.Fatal("cli.Dependencies must be a struct")
					}
					for _, field := range structure.Fields.List {
						fn, ok := field.Type.(*ast.FuncType)
						if !ok || fn.Results == nil || len(fn.Results.List) == 0 {
							continue
						}
						if result, ok := fn.Results.List[0].Type.(*ast.Ident); ok {
							contractNames[result.Name] = true
						}
					}
				}
			}
		}
	}

	contracts := make(map[string]map[string]bool)
	for name := range contractNames {
		methods := make(map[string]bool)
		collectInterfaceMethods(name, interfaces, methods, make(map[string]bool))
		delete(methods, "Close")
		if len(methods) > 0 {
			contracts[name] = methods
		}
	}
	return contracts
}

func collectInterfaceMethods(name string, interfaces map[string]*ast.InterfaceType, methods, visiting map[string]bool) {
	if visiting[name] {
		return
	}
	visiting[name] = true
	iface := interfaces[name]
	if iface == nil {
		return
	}
	for _, field := range iface.Methods.List {
		if len(field.Names) > 0 {
			for _, method := range field.Names {
				methods[method.Name] = true
			}
			continue
		}
		if embedded, ok := field.Type.(*ast.Ident); ok {
			collectInterfaceMethods(embedded.Name, interfaces, methods, visiting)
		}
	}
}

func receiverName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	default:
		return ""
	}
}

func implementsSemanticMethods(got, required map[string]bool) bool {
	for method := range required {
		if !got[method] {
			return false
		}
	}
	return len(required) > 0
}
