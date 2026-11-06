package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttendanceContextIsImplementedWithoutProviderLeakage(t *testing.T) {
	model, err := os.ReadFile("../../docs/domain/model.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(model)
	for _, marker := range []string{"id: attendance", "status: implemented", "package: internal/domain/attendance", "workforce-attendance"} {
		if !strings.Contains(text, marker) {
			t.Errorf("Domain model missing Attendance marker %q", marker)
		}
	}

	err = filepath.Walk("../../internal/domain/attendance", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, forbidden := range []string{"firestore", "secretmanager", "net/http", "liff", "internal/adapter", "internal/application"} {
			if strings.Contains(strings.ToLower(string(data)), forbidden) {
				t.Errorf("Attendance Domain %s contains provider/transport term %q", path, forbidden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStatelessLocationIntakeWasRetired(t *testing.T) {
	if _, err := os.Stat("../../internal/application/usecases/location-intake"); !os.IsNotExist(err) {
		t.Fatalf("location-intake compatibility package still exists: %v", err)
	}
	for _, path := range []string{
		"../../api/openapi/miniapp.yaml",
		"../../docs/architecture/overview.md",
		"../../docs/line/miniapp-attendance.md",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "/location-validations") {
			t.Errorf("%s retains retired location validation route", path)
		}
	}
}
