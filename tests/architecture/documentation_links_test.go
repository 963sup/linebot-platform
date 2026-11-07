package architecture

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var markdownLink = regexp.MustCompile(`!?\[[^]]*\]\(([^)]+)\)`)

func TestArchitectureAuthorityAndRelativeLinks(t *testing.T) {
	root := filepath.Join("..", "..")
	obsoleteDesign := "DESIGN" + ".md"
	for _, obsolete := range []string{
		obsoleteDesign,
		filepath.Join("docs", "architecture", "layer-first-blueprint.md"),
		filepath.Join("docs", "architecture", "boundaries.md"),
		filepath.Join("docs", "architecture", "dependency-rules.md"),
	} {
		if _, err := os.Stat(filepath.Join(root, obsolete)); !os.IsNotExist(err) {
			t.Errorf("obsolete architecture source still exists: %s", obsolete)
		}
	}

	paths := []string{filepath.Join(root, "README.md"), filepath.Join(root, "AGENTS.md")}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	authorities := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Error(err)
			continue
		}
		text := string(data)
		if strings.Contains(text, "本文件是本專案唯一的架構與依賴規範") {
			authorities++
			if filepath.Clean(path) != filepath.Join(root, "docs", "architecture", "blueprint.md") {
				t.Errorf("architecture authority declared outside blueprint: %s", path)
			}
		}
		if strings.Contains(text, obsoleteDesign) {
			t.Errorf("obsolete root architecture reference remains in %s", path)
		}
		for _, match := range markdownLink.FindAllStringSubmatch(text, -1) {
			target := strings.Trim(strings.Fields(match[1])[0], "<>")
			if target == "" || strings.HasPrefix(target, "#") || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			target, _, _ = strings.Cut(target, "#")
			if target == "" {
				continue
			}
			// Skills are host-provided development guidance, not source required
			// by the production build. Their files are intentionally excluded from
			// Cloud Build uploads, so do not make this runtime-source test depend
			// on a local agent installation.
			if strings.HasPrefix(target, ".agents/") {
				continue
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(target)))
			if _, err := os.Stat(resolved); err != nil {
				t.Errorf("%s has broken relative link %q: %v", path, target, err)
			}
		}
	}
	if authorities != 1 {
		t.Fatalf("architecture authority declarations = %d, want 1", authorities)
	}
}
