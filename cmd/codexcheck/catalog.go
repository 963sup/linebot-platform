package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// sdkCatalog is a routing inventory, not an API inventory or a second version lockfile.
type sdkCatalog struct {
	SchemaVersion int         `toml:"schema_version"`
	Modules       []sdkModule `toml:"modules"`
}

type sdkModule struct {
	Module       string   `toml:"module"`
	Provider     string   `toml:"provider"`
	Kind         string   `toml:"kind"`
	Status       string   `toml:"status"`
	Imports      []string `toml:"imports"`
	Capabilities []string `toml:"capabilities"`
	Factory      string   `toml:"factory"`
	Adapter      string   `toml:"adapter"`
	Reference    string   `toml:"reference"`
}

// directModules reads standard go.mod require directives without resolving modules
// or invoking Go, so catalog validation works offline and does not edit go.sum.
// The Go toolchain remains authoritative for resolving the full dependency graph.
func directModules(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	modules := make(map[string]string)
	inBlock := false
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "require (" {
			if inBlock {
				return nil, fmt.Errorf("nested require block in %s", file)
			}
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if !inBlock {
			if !strings.HasPrefix(line, "require ") {
				continue
			}
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		}
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		spec, comment, _ := strings.Cut(line, "//")
		if strings.TrimSpace(comment) == "indirect" {
			continue
		}
		fields := strings.Fields(spec)
		if len(fields) != 2 {
			return nil, fmt.Errorf("unsupported require directive in %s: %q", file, line)
		}
		if _, exists := modules[fields[0]]; exists {
			return nil, fmt.Errorf("duplicate direct module %s", fields[0])
		}
		modules[fields[0]] = fields[1]
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	if inBlock {
		return nil, fmt.Errorf("unclosed require block in %s", file)
	}
	return modules, nil
}

func validDirectory(dir, prefix string) bool {
	return dir != "" && path.Clean(dir) == dir &&
		strings.HasPrefix(dir, prefix+"/") &&
		!strings.Contains(dir, "\\")
}

func hasGoSource(root, dir string) bool {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") &&
			!strings.HasSuffix(entry.Name(), "_test.go") {
			return true
		}
	}
	return false
}

func validateCatalog(root string) error {
	direct, err := directModules(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	var catalog sdkCatalog
	if err := readTOML(filepath.Join(root, "docs", "sdk", "catalog.toml"), &catalog); err != nil {
		return err
	}
	if catalog.SchemaVersion != 1 || len(catalog.Modules) == 0 {
		return fmt.Errorf("SDK catalog must have schema_version = 1 and at least one module")
	}

	seen := make(map[string]bool, len(catalog.Modules))
	for _, m := range catalog.Modules {
		if m.Module == "" || seen[m.Module] {
			return fmt.Errorf("SDK catalog contains an empty or duplicate module: %q", m.Module)
		}
		seen[m.Module] = true
		if _, ok := direct[m.Module]; !ok {
			return fmt.Errorf("SDK catalog module %s is not a direct go.mod dependency", m.Module)
		}
		if strings.TrimSpace(m.Provider) == "" {
			return fmt.Errorf("%s: provider is required", m.Module)
		}
		if m.Kind != "service" && m.Kind != "utility" && m.Kind != "tooling" {
			return fmt.Errorf("%s: invalid kind %q", m.Module, m.Kind)
		}
		if m.Status != "integrated" && m.Status != "declared" {
			return fmt.Errorf("%s: invalid status %q", m.Module, m.Status)
		}
		parsed, err := url.Parse(m.Reference)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("%s: reference must be an HTTPS URL", m.Module)
		}
		if len(m.Imports) == 0 || len(m.Capabilities) == 0 {
			return fmt.Errorf("%s: imports and capabilities are required", m.Module)
		}
		imports := make(map[string]bool)
		for _, imp := range m.Imports {
			if imp != m.Module && !strings.HasPrefix(imp, m.Module+"/") {
				return fmt.Errorf("%s: import %q is outside module", m.Module, imp)
			}
			if imports[imp] {
				return fmt.Errorf("%s: duplicate import %q", m.Module, imp)
			}
			imports[imp] = true
		}
		tags := make(map[string]bool)
		for _, tag := range m.Capabilities {
			if strings.TrimSpace(tag) == "" || tags[tag] {
				return fmt.Errorf("%s: empty or duplicate capability %q", m.Module, tag)
			}
			tags[tag] = true
		}
		if m.Kind == "service" && !validDirectory(m.Factory, "internal/infrastructure") {
			return fmt.Errorf("%s: service factory must reside in internal/infrastructure", m.Module)
		}
		if m.Factory != "" && !validDirectory(m.Factory, "internal/infrastructure") {
			return fmt.Errorf("%s: invalid factory directory", m.Module)
		}
		if m.Adapter != "" && !validDirectory(m.Adapter, "internal/adapter/outbound") {
			return fmt.Errorf("%s: invalid outbound adapter directory", m.Module)
		}
		if m.Status == "integrated" && m.Kind == "service" {
			if !hasGoSource(root, m.Factory) {
				return fmt.Errorf("%s: integrated factory has no Go implementation", m.Module)
			}
			if m.Adapter == "" || !hasGoSource(root, m.Adapter) {
				return fmt.Errorf("%s: integrated service has no outbound adapter implementation", m.Module)
			}
		}
	}
	if len(seen) != len(direct) {
		var missing []string
		for module := range direct {
			if !seen[module] {
				missing = append(missing, module)
			}
		}
		slices.Sort(missing)
		return fmt.Errorf("SDK catalog is missing direct go.mod modules: %v", missing)
	}
	return nil
}
