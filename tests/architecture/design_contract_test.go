package architecture

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"image/png"
	"io"
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
		"../assets/architecture/go-ddd-hexagonal-architecture-overview.png",
		"../assets/architecture/go-ddd-hexagonal-architecture-overview.svg",
		"../assets/architecture/go-ddd-hexagonal-architecture-codex.svg",
		"../assets/architecture/go-ddd-hexagonal-architecture-engineering.svg",
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

	imagePath := filepath.Join(root, "docs", "assets", "architecture", "go-ddd-hexagonal-architecture-overview.png")
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
	if config.Width != 2400 || config.Height != 1650 {
		t.Fatalf("architecture overview asset size = %dx%d, want 2400x1650", config.Width, config.Height)
	}
	assertSHA256(t, imageData, "8b6bff2e44ca6fd55475e8a90055f76064fc9c1c024ddfdd57c00876af348dad")

	assets := []struct {
		name                   string
		hash                   string
		width, height, viewBox string
		required               []string
	}{
		{
			name: "go-ddd-hexagonal-architecture-overview.svg", hash: "89d2fbd36fd9677a78ee6882871e8ec36d73dea540f0ec32e0de697433753a5d",
			width: "1600", height: "1100", viewBox: "0 0 1600 1100",
			required: []string{"非規範性概覽", "Staging/Production Not Verified", "dependency 指向 consumer"},
		},
		{
			name: "go-ddd-hexagonal-architecture-codex.svg", hash: "64f86a5bff8622729284adeb029f82778d2cc8e10851142f9511c6bb2213485b",
			width: "3600", height: "2560", viewBox: "0 0 3600 2560",
			required: []string{"非規範性參考圖", "Layer-first Modular Monolith", "internal/bootstrap"},
		},
		{
			name: "go-ddd-hexagonal-architecture-engineering.svg", hash: "1752116505ddd35913a678eb429871d73688bd6bdac21485aab2b84cf3e5197a",
			width: "3200px", height: "2100px", viewBox: "0 0 3200 2100",
			required: []string{"非規範性", "Layer-first Structure", "Staging/Production Not Verified", "unknown / manual reconciliation"},
		},
	}
	for _, asset := range assets {
		t.Run(asset.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, "docs", "assets", "architecture", asset.name))
			if err != nil {
				t.Fatal(err)
			}
			assertSHA256(t, data, asset.hash)
			validateArchitectureSVG(t, data, asset.width, asset.height, asset.viewBox)
			text := string(data)
			for _, marker := range asset.required {
				if !strings.Contains(text, marker) {
					t.Errorf("%s is missing semantic marker %q", asset.name, marker)
				}
			}
			for _, marker := range []string{"Codex 可執行架構藍圖", "示意是規範性概念圖", "Production-Ready", "cmd/server/main.go", "internal/&lt;context&gt;"} {
				if strings.Contains(text, marker) {
					t.Errorf("%s contains obsolete or misleading marker %q", asset.name, marker)
				}
			}
		})
	}
}

func assertSHA256(t *testing.T, data []byte, want string) {
	t.Helper()
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
		t.Fatalf("architecture blueprint asset SHA-256 = %s, want %s", got, want)
	}
}

func validateArchitectureSVG(t *testing.T, data []byte, width, height, viewBox string) {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rootSeen := false
	titleSeen := false
	descSeen := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("architecture blueprint SVG is invalid XML: %v", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if !rootSeen {
			rootSeen = true
			if start.Name.Local != "svg" || start.Name.Space != "http://www.w3.org/2000/svg" {
				t.Fatalf("architecture blueprint root = {%s}%s, want SVG namespace", start.Name.Space, start.Name.Local)
			}
			attrs := make(map[string]string, len(start.Attr))
			for _, attr := range start.Attr {
				attrs[attr.Name.Local] = attr.Value
			}
			for name, want := range map[string]string{"width": width, "height": height, "viewBox": viewBox, "role": "img", "lang": "zh-Hant"} {
				if attrs[name] != want {
					t.Fatalf("architecture blueprint SVG %s = %q, want %q", name, attrs[name], want)
				}
			}
			if strings.TrimSpace(attrs["aria-label"]) == "" {
				t.Fatal("architecture blueprint SVG must have a non-empty aria-label")
			}
		}
		if start.Name.Local == "title" {
			titleSeen = true
		}
		if start.Name.Local == "desc" {
			descSeen = true
		}
		if start.Name.Local == "script" || start.Name.Local == "foreignObject" {
			t.Fatalf("architecture blueprint SVG must not contain active element %q", start.Name.Local)
		}
		for _, attr := range start.Attr {
			if strings.HasPrefix(strings.ToLower(attr.Name.Local), "on") {
				t.Fatalf("architecture blueprint SVG must not contain event handler %q", attr.Name.Local)
			}
			if attr.Name.Local != "href" {
				continue
			}
			value := strings.ToLower(strings.TrimSpace(attr.Value))
			if strings.HasPrefix(value, "http:") || strings.HasPrefix(value, "https:") || strings.HasPrefix(value, "//") || strings.HasPrefix(value, "javascript:") || strings.HasPrefix(value, "data:") {
				t.Fatalf("architecture blueprint SVG contains external or active href %q", attr.Value)
			}
		}
	}
	if !rootSeen {
		t.Fatal("architecture blueprint SVG has no root element")
	}
	if !titleSeen || !descSeen {
		t.Fatal("architecture blueprint SVG must contain title and desc metadata")
	}
}
