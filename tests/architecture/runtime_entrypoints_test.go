package architecture

import (
	"go/ast"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRuntimeEntrypointsOnlyEnterThroughBootstrap(t *testing.T) {
	cmdRoot := "../../cmd"
	entries, err := os.ReadDir(cmdRoot)
	if err != nil {
		t.Fatal(err)
	}
	runtimeCommands := map[string]bool{"webhook": true, "worker": true, "ops": true, "miniapp": true}
	toolCommands := map[string]bool{"codexcheck": true, "codexhandoff": true, "codexplugins": true, "configcheck": true, "domainmodel": true}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if toolCommands[name] {
			continue
		}
		if !runtimeCommands[name] {
			t.Errorf("cmd/%s is not classified as a runtime entrypoint or repository tool", name)
			continue
		}
		root := filepath.Join(cmdRoot, name)
		walkProductionGoFiles(t, root, func(path string, file *ast.File) {
			for _, imp := range file.Imports {
				value, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Errorf("%s: invalid import %s: %v", path, imp.Path.Value, err)
					continue
				}
				if isThirdParty(value) {
					t.Errorf("%s: runtime entrypoint imports third-party package %q instead of entering through bootstrap", path, value)
					continue
				}
				if strings.HasPrefix(value, modulePath+"/") && value != internalPath+"bootstrap" {
					t.Errorf("%s: runtime entrypoint imports %q instead of entering through bootstrap", path, value)
				}
			}
		})
	}
}
