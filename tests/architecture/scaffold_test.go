package architecture

import (
	"io/fs"
	"path/filepath"
	"testing"
)

func TestInternalTreeHasNoScaffoldMarkers(t *testing.T) {
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && entry.Name() == ".gitkeep" {
			t.Errorf("obsolete internal scaffold marker: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
