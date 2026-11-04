package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
)

func TestWorkspacePoliciesMatchPinnedDiscovery(t *testing.T) {
	all := workspace.Supported()
	if len(all) != 62 {
		t.Fatalf("got %d static operation policies, want 62", len(all))
	}
	found := map[string]workspace.Policy{}
	for _, p := range all {
		found[p.API+"."+p.Operation] = p
	}
	for _, name := range []string{"calendar", "tasks", "forms"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "docs", "google", name+"-endpoints.json"))
		if err != nil {
			t.Fatal(err)
		}
		var inv workspaceInventory
		if err := json.Unmarshal(data, &inv); err != nil {
			t.Fatal(err)
		}
		for _, m := range inv.Methods {
			p, ok := found[name+"."+m.Operation]
			if !ok || p.Mode != m.Mode {
				t.Fatalf("static policy mismatch %s %s", name, m.Operation)
			}
		}
	}
}
