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
		"../assets/architecture/go-ddd-hexagonal-architecture-blueprint.png",
		"../assets/architecture/go-ddd-hexagonal-architecture-blueprint.svg",
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
	assertSHA256(t, imageData, "ef0d102f2ca863586c27251e5f4848da66426669384dd2a323daa1db4de26d5e")

	svgPath := filepath.Join(root, "docs", "assets", "architecture", "go-ddd-hexagonal-architecture-blueprint.svg")
	svgData, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatal(err)
	}
	assertSHA256(t, svgData, "e546c4d23e2b53e600aadbd0926fbeab768606e171d94f70c87854e64a3ce148")
	validateArchitectureSVG(t, svgData)
}

func assertSHA256(t *testing.T, data []byte, want string) {
	t.Helper()
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
		t.Fatalf("architecture blueprint asset SHA-256 = %s, want %s", got, want)
	}
}

func validateArchitectureSVG(t *testing.T, data []byte) {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rootSeen := false
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
			for name, want := range map[string]string{"width": "3600", "height": "2560", "viewBox": "0 0 3600 2560"} {
				if attrs[name] != want {
					t.Fatalf("architecture blueprint SVG %s = %q, want %q", name, attrs[name], want)
				}
			}
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
}
