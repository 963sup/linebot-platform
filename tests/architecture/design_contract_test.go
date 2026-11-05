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
	assertSHA256(t, imageData, "1da26ba56535dc3c7f832f88e331643ca935d648fd13e2dcca81841f3aa8b858")

	assets := []struct {
		name                   string
		hash                   string
		width, height, viewBox string
		required               []string
	}{
		{
			name: "go-ddd-hexagonal-architecture-overview.svg", hash: "707bc4c7965495d025251e396fd5f1d7c563e1969b37b0966fbee1c6fb4b9037",
			width: "1600", height: "1100", viewBox: "0 0 1600 1100",
			required: []string{"非規範性概覽", "Staging/Production Not Verified", "dependency 指向 consumer"},
		},
		{
			name: "go-ddd-hexagonal-architecture-codex.svg", hash: "a842870ac4ba02d343e752bfd81e4fbb4ee103fd9eca9b5d1fedfd904b6ceaa6",
			width: "3600", height: "2560", viewBox: "0 0 3600 2560",
			required: []string{"非規範性參考圖", "Layer-first Modular Monolith", "internal/bootstrap", "classified-safe retry", "unknown / manual reconciliation", "Application capability ports；Domain repository 由 Domain 擁有"},
		},
		{
			name: "go-ddd-hexagonal-architecture-engineering.svg", hash: "c1ab4cd93a8f550cca61818b973f70caecb578e3310da1e31a8a3da178091233",
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
			groups := architectureSVGGroupText(t, data)
			ownership := groups["ownership-contracts"]
			for _, marker := range []string{"Application capability port", "Domain repository"} {
				if !strings.Contains(ownership, marker) {
					t.Errorf("ownership-contracts group is missing %q", marker)
				}
			}
			for _, obsolete := range []string{"Driven Adapters（實作 Outbound Ports", "符合 Application Port", "Application owns Outbound Ports"} {
				if strings.Contains(text, obsolete) {
					t.Errorf("%s contains obsolete absolute ownership sentence %q", asset.name, obsolete)
				}
			}
			if asset.name != "go-ddd-hexagonal-architecture-engineering.svg" {
				content := groups["architecture-content"]
				for _, marker := range asset.required {
					if !strings.Contains(content, marker) {
						t.Errorf("architecture-content group is missing %q", marker)
					}
				}
			}
			if asset.name == "go-ddd-hexagonal-architecture-engineering.svg" {
				quality := groups["quality-targets"]
				for _, marker := range []string{"Quality Targets", "Staging/Production Not Verified", "Classified-safe retries", "unknown / manual reconciliation"} {
					if !strings.Contains(quality, marker) {
						t.Errorf("quality-targets group is missing %q", marker)
					}
				}
				if strings.Contains(quality, "✓") || !strings.Contains(quality, "•") {
					t.Error("quality-targets must use neutral bullets, not completion checks")
				}
				ownership := groups["tactical-design"]
				for _, marker := range []string{"Application capability port 由 consumer 擁有", "Domain repository 可由 Domain 擁有"} {
					if !strings.Contains(ownership, marker) {
						t.Errorf("tactical-design ownership group is missing %q", marker)
					}
				}
			}
		})
	}
}

func TestArchitectureSVGRejectsUnsafeContent(t *testing.T) {
	const open = `<svg xmlns="http://www.w3.org/2000/svg" xml:lang="zh-Hant" role="img" aria-labelledby="t d" width="1" height="1" viewBox="0 0 1 1"><title id="t">title</title><desc id="d">description</desc>`
	const close = `</svg>`
	tests := map[string]string{
		"doctype":           `<!DOCTYPE svg>` + open + close,
		"entity":            `<!DOCTYPE svg [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>` + open + close,
		"stylesheet PI":     `<?xml-stylesheet type="text/css" href="https://example.invalid/a.css"?>` + open + close,
		"image":             open + `<image href="#local"/>` + close,
		"use":               open + `<use href="#local"/>` + close,
		"style import":      open + `<style>@import "https://example.invalid/a.css";</style>` + close,
		"style http":        open + `<style>rect{fill:url(http://example.invalid/a)}</style>` + close,
		"style https quote": open + `<style>rect{fill:url("https://example.invalid/a")}</style>` + close,
		"style protocol":    open + `<style>rect{fill:url(//example.invalid/a)}</style>` + close,
		"style data":        open + `<style>rect{fill:url(data:image/svg+xml,x)}</style>` + close,
		"style javascript":  open + `<style>rect{fill:url(javascript:alert(1))}</style>` + close,
		"external attr":     open + `<rect fill="https://example.invalid/a"/>` + close,
		"metadata order":    `<svg xmlns="http://www.w3.org/2000/svg" xml:lang="zh-Hant" role="img" aria-labelledby="t d" width="1" height="1" viewBox="0 0 1 1"><desc id="d">description</desc><title id="t">title</title></svg>`,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateArchitectureSVGData([]byte(input), "1", "1", "0 0 1 1"); err == nil {
				t.Fatal("unsafe SVG was accepted")
			}
		})
	}
}

func architectureSVGGroupText(t *testing.T, data []byte) map[string]string {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(data))
	groups := make(map[string]*strings.Builder)
	var active []string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode architecture SVG groups: %v", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "g" {
				id := ""
				for _, attr := range value.Attr {
					if attr.Name.Local == "id" {
						id = attr.Value
					}
				}
				active = append(active, id)
				if id != "" && groups[id] == nil {
					groups[id] = &strings.Builder{}
				}
			}
		case xml.CharData:
			for _, id := range active {
				if id != "" {
					groups[id].Write(value)
				}
			}
		case xml.EndElement:
			if value.Name.Local == "g" && len(active) > 0 {
				active = active[:len(active)-1]
			}
		}
	}
	result := make(map[string]string, len(groups))
	for id, builder := range groups {
		result[id] = builder.String()
	}
	return result
}

func assertSHA256(t *testing.T, data []byte, want string) {
	t.Helper()
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
		t.Fatalf("architecture blueprint asset SHA-256 = %s, want %s", got, want)
	}
}

func validateArchitectureSVG(t *testing.T, data []byte, width, height, viewBox string) {
	t.Helper()
	if err := validateArchitectureSVGData(data, width, height, viewBox); err != nil {
		t.Fatal(err)
	}
}

func validateArchitectureSVGData(data []byte, width, height, viewBox string) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rootSeen := false
	var rootTitleID, rootDescID string
	var titleText, descText strings.Builder
	depth := 0
	rootChild := 0
	currentRootMetadata := ""
	insideStyle := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("architecture blueprint SVG is invalid XML: %w", err)
		}
		if directive, ok := token.(xml.Directive); ok {
			return fmt.Errorf("architecture blueprint SVG must not contain XML directive %q", directive)
		}
		if instruction, ok := token.(xml.ProcInst); ok {
			if instruction.Target != "xml" {
				return fmt.Errorf("architecture blueprint SVG must not contain processing instruction %q", instruction.Target)
			}
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			if charData, ok := token.(xml.CharData); ok {
				if currentRootMetadata != "" {
					switch currentRootMetadata {
					case "title":
						titleText.Write(charData)
					case "desc":
						descText.Write(charData)
					}
				}
				if insideStyle && unsafeSVGReference(string(charData)) {
					return fmt.Errorf("architecture blueprint SVG style contains external import or URL")
				}
			}
			if end, ok := token.(xml.EndElement); ok {
				if end.Name.Local == "style" {
					insideStyle = false
				}
				if depth == 2 && (end.Name.Local == "title" || end.Name.Local == "desc") {
					currentRootMetadata = ""
				}
				depth--
			}
			continue
		}
		depth++
		if !rootSeen {
			rootSeen = true
			if start.Name.Local != "svg" || start.Name.Space != "http://www.w3.org/2000/svg" {
				return fmt.Errorf("architecture blueprint root = {%s}%s, want SVG namespace", start.Name.Space, start.Name.Local)
			}
			attrs := make(map[string]string, len(start.Attr))
			for _, attr := range start.Attr {
				attrs[attr.Name.Local] = attr.Value
			}
			for name, want := range map[string]string{"width": width, "height": height, "viewBox": viewBox, "role": "img", "lang": "zh-Hant"} {
				if attrs[name] != want {
					return fmt.Errorf("architecture blueprint SVG %s = %q, want %q", name, attrs[name], want)
				}
			}
			labelledBy := strings.Fields(attrs["aria-labelledby"])
			if len(labelledBy) != 2 || labelledBy[0] == labelledBy[1] {
				return fmt.Errorf("architecture blueprint SVG must reference distinct title and desc IDs with aria-labelledby")
			}
			rootTitleID, rootDescID = labelledBy[0], labelledBy[1]
		}
		if depth == 2 {
			rootChild++
			if (rootChild == 1 && start.Name.Local != "title") || (rootChild == 2 && start.Name.Local != "desc") {
				return fmt.Errorf("architecture blueprint SVG title and desc must be the first two root children")
			}
		}
		if depth == 2 && (start.Name.Local == "title" || start.Name.Local == "desc") {
			var id string
			for _, attr := range start.Attr {
				if attr.Name.Local == "id" {
					id = attr.Value
				}
			}
			if start.Name.Local == "title" && id != rootTitleID {
				return fmt.Errorf("root title id = %q, want %q", id, rootTitleID)
			}
			if start.Name.Local == "desc" && id != rootDescID {
				return fmt.Errorf("root desc id = %q, want %q", id, rootDescID)
			}
			currentRootMetadata = start.Name.Local
		}
		if start.Name.Local == "script" || start.Name.Local == "foreignObject" || start.Name.Local == "image" || start.Name.Local == "use" {
			return fmt.Errorf("architecture blueprint SVG must not contain active or referencing element %q", start.Name.Local)
		}
		if start.Name.Local == "style" {
			insideStyle = true
		}
		for _, attr := range start.Attr {
			if attr.Name.Local == "xmlns" || attr.Name.Space == "xmlns" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(attr.Name.Local), "on") {
				return fmt.Errorf("architecture blueprint SVG must not contain event handler %q", attr.Name.Local)
			}
			if unsafeSVGReference(attr.Value) {
				return fmt.Errorf("architecture blueprint SVG contains external or active reference %q", attr.Value)
			}
		}
	}
	if !rootSeen {
		return fmt.Errorf("architecture blueprint SVG has no root element")
	}
	if strings.TrimSpace(titleText.String()) == "" || strings.TrimSpace(descText.String()) == "" {
		return fmt.Errorf("architecture blueprint SVG must contain direct non-empty title and desc metadata")
	}
	return nil
}

func unsafeSVGReference(value string) bool {
	value = strings.ToLower(strings.Join(strings.Fields(value), ""))
	value = strings.NewReplacer(`"`, "", `'`, "").Replace(value)
	return strings.Contains(value, "@import") || strings.Contains(value, "url(http:") || strings.Contains(value, "url(https:") ||
		strings.Contains(value, "url(//") || strings.Contains(value, "url(data:") || strings.Contains(value, "url(javascript:") ||
		strings.HasPrefix(value, "http:") || strings.HasPrefix(value, "https:") || strings.HasPrefix(value, "//") ||
		strings.HasPrefix(value, "data:") || strings.HasPrefix(value, "javascript:")
}
