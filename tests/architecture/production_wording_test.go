package architecture

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var productionReadinessClaim = regexp.MustCompile(`(?i)production[- ]read(?:y|iness)`)
var productionClauseBreak = regexp.MustCompile(`(?i);\s*|[!?。；！？|，]|,\s+|\.\s+|\bbut\b|\bhowever\b|但|然而`)

var productionReadinessGuards = []string{
	"not production-ready",
	"not proof",
	"does not imply",
	"none of these statuses",
	"cannot claim",
	"do not mark",
	"require staging acceptance",
	"before any production readiness statement",
	"仍是未正式部署",
	"不是 production-ready",
	"不能宣稱 production-ready",
	"不能稱為 production-ready",
	"不能因 ci 通過宣稱 production-ready",
	"仍未完成",
	"完成 `docs/operations/staging-acceptance.md` 每一項 gate",
}

// TestDocumentationDoesNotClaimProductionReadiness keeps repository-local
// verification distinct from authorized staging and production evidence.
func TestDocumentationDoesNotClaimProductionReadiness(t *testing.T) {
	root := filepath.Join("..", "..")
	paths := []string{
		filepath.Join(root, "README.md"),
	}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range paths {
		checkProductionWording(t, root, path)
	}
}

func checkProductionWording(t *testing.T, root, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Error(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		for _, clause := range unguardedProductionReadinessClauses(line) {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}
			t.Errorf("%s:%d makes an unqualified production-readiness claim: %s", filepath.ToSlash(rel), lineNumber, clause)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Error(err)
	}
}

func unguardedProductionReadinessClauses(line string) []string {
	var unguarded []string
	for _, clause := range productionClauseBreak.Split(line, -1) {
		clause = strings.TrimSpace(clause)
		if !productionReadinessClaim.MatchString(clause) {
			continue
		}
		lower := strings.ToLower(clause)
		guarded := false
		for _, marker := range productionReadinessGuards {
			if strings.Contains(lower, marker) {
				guarded = true
				break
			}
		}
		if !guarded {
			unguarded = append(unguarded, clause)
		}
	}
	return unguarded
}

func TestProductionReadinessWordingRequiresSameClauseGuard(t *testing.T) {
	tests := []struct {
		name string
		line string
		want int
	}{
		{name: "direct negative", line: "This service is not production-ready."},
		{name: "implication negative", line: "CI success does not imply production readiness."},
		{name: "chinese negative", line: "目前不能宣稱 production-ready。"},
		{name: "bare claim", line: "The service is production-ready.", want: 1},
		{name: "guard in prior clause", line: "This is not proof; the service is production-ready.", want: 1},
		{name: "guard after contrasting claim", line: "The service is production-ready, but it is not production-ready.", want: 1},
		{name: "chinese guard in another clause", line: "目前仍未完成；下一版將是 production-ready。", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(unguardedProductionReadinessClauses(tt.line)); got != tt.want {
				t.Fatalf("unguardedProductionReadinessClauses(%q) = %d, want %d", tt.line, got, tt.want)
			}
		})
	}
}
