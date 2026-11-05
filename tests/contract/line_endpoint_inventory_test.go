package contract

import (
	"path/filepath"
	"testing"

	"github.com/963sup/linebot-platform/internal/governance/domainmodel"
)

// The pinned LINE OpenAPI snapshot remains an external-fact inventory. Domain
// classification is validated separately through the capability map.
func TestLINEEndpointInventory(t *testing.T) {
	root := filepath.Join("..", "..")
	inventory, err := domainmodel.LoadEndpointInventory(filepath.Join(root, "docs", "line", "endpoint-inventory.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Endpoints) != domainmodel.ExpectedEndpointCount {
		t.Fatalf("inventory count = %d, want %d", len(inventory.Endpoints), domainmodel.ExpectedEndpointCount)
	}
	for _, endpoint := range inventory.Endpoints {
		if endpoint.Integration == "integrated_offline" && endpoint.Test != "contract" {
			t.Errorf("%s: offline integration lacks contract evidence", endpoint.Operation)
		}
		if endpoint.Integration == "not_integrated" && endpoint.Test != "none" {
			t.Errorf("%s: unintegrated endpoint claims a test", endpoint.Operation)
		}
	}
}
