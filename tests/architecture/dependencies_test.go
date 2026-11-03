package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDependencyDirection(t *testing.T) {
	for _, layer := range []struct {
		dir          string
		allowedLocal string
	}{
		{"../../internal/domain", "github.com/963sup/linebot-platform/internal/domain"},
		{"../../internal/application", "github.com/963sup/linebot-platform/internal/application"},
	} {
		err := filepath.WalkDir(layer.dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range f.Imports {
				p, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				if strings.HasPrefix(p, "github.com/963sup/linebot-platform/internal/") {
					if !strings.HasPrefix(p, layer.allowedLocal) &&
						!(layer.dir == "../../internal/application" && strings.HasPrefix(p, "github.com/963sup/linebot-platform/internal/domain")) {
						t.Errorf("%s must not depend on %s", path, p)
					}
				} else if strings.Contains(strings.Split(p, "/")[0], ".") {
					t.Errorf("domain/application external dependency: %s imports %s", path, p)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
