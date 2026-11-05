package architecture

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDesignContract protects the documented architecture vocabulary and views
// that contributors use when extending the repository.
func TestDesignContract(t *testing.T) {
	root := filepath.Join("..", "..")
	blueprintPath := filepath.Join(root, "docs", "architecture", "blueprint.md")
	content, err := os.ReadFile(blueprintPath)
	if err != nil {
		t.Fatal(err)
	}
	design := string(content)

	required := []string{
		"internal/application/usecases/<capability>/",
		"capability-local consumer-owned interfaces",
		"tests/{architecture,contract,integration,performance,e2e}",
		"sequenceDiagram",
		"flowchart LR",
		"Hexagonal Architecture 視圖",
		"DDD Tactical Building Blocks",
		"classDiagram",
		"stateDiagram-v2",
		"Event Aggregate 狀態圖",
		"AccountLink Aggregate 狀態圖",
		"Interface implementation",
		"Domain Event 與 Integration Event",
		"Production baseline implemented; Staging/Production not verified.",
		"../assets/architecture/go-ddd-hexagonal-architecture-blueprint.png",
		"pending",
		"sending",
		"sent",
		"expired",
		"unknown",
	}
	for _, marker := range required {
		if !strings.Contains(design, marker) {
			t.Errorf("architecture blueprint is missing required marker %q", marker)
		}
	}

	forbidden := []string{
		"internal/application/ports/inbound/",
		"internal/application/ports/outbound/",
		"internal/ports/inbound",
		"internal/ports/outbound",
		"internal/adapters/",
		"test/performance",
		"test/e2e",
	}
	for _, marker := range forbidden {
		if strings.Contains(design, marker) {
			t.Errorf("architecture blueprint contains obsolete marker %q", marker)
		}
	}

	imagePath := filepath.Join(root, "docs", "assets", "architecture", "go-ddd-hexagonal-architecture-blueprint.png")
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(imageData, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatal("architecture blueprint asset is not a PNG")
	}
	config, err := png.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width < 1000 || config.Height < 600 {
		t.Fatalf("architecture blueprint asset is unexpectedly small: %dx%d", config.Width, config.Height)
	}
}
