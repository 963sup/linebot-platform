// Command domainmodel validates Strategic DDD manifests and manages their
// deterministic human-readable projection.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/963sup/linebot-platform/internal/governance/domainmodel"
)

func main() {
	mode := flag.String("mode", "check", "check or write the generated context map")
	flag.Parse()
	if err := run(*mode); err != nil {
		fmt.Fprintln(os.Stderr, "domain model:", err)
		os.Exit(1)
	}
}

func run(mode string) error {
	if mode != "check" && mode != "write" {
		return fmt.Errorf("unsupported mode %q", mode)
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	inventory, model, capabilities, dependencies, err := load(root)
	if err != nil {
		return err
	}
	if err := domainmodel.ValidateRepository(root, inventory, model, capabilities, dependencies); err != nil {
		return err
	}
	generated, err := domainmodel.RenderContextMap(model, capabilities, dependencies)
	if err != nil {
		return err
	}
	path := filepath.Join(root, "docs", "domain", "context-map.md")
	switch mode {
	case "write":
		return os.WriteFile(path, generated, 0o644)
	case "check":
		actual, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(actual, generated) {
			return fmt.Errorf("%s is stale; run go run ./cmd/domainmodel -mode=write", filepath.ToSlash(path))
		}
		return nil
	}
	return nil
}

func load(root string) (*domainmodel.EndpointInventory, *domainmodel.DomainModel, *domainmodel.CapabilityMap, *domainmodel.DependencyCapabilities, error) {
	inventory, err := domainmodel.LoadEndpointInventory(filepath.Join(root, "docs", "line", "endpoint-inventory.yaml"))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	model, err := domainmodel.LoadDomainModel(filepath.Join(root, "docs", "domain", "model.yaml"))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	capabilities, err := domainmodel.LoadCapabilityMap(filepath.Join(root, "docs", "domain", "capability-map.yaml"))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	dependencies, err := domainmodel.LoadDependencyCapabilities(filepath.Join(root, "docs", "domain", "dependency-capabilities.yaml"))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return inventory, model, capabilities, dependencies, nil
}
