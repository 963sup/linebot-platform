package architecture

import (
	"os"
	"strings"
	"testing"
)

// TestDesignContract protects the documented architecture vocabulary and views
// that contributors use when extending the repository.
func TestDesignContract(t *testing.T) {
	content, err := os.ReadFile("../../DESIGN.md")
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
		"pending",
		"sending",
		"sent",
		"expired",
		"unknown",
	}
	for _, marker := range required {
		if !strings.Contains(design, marker) {
			t.Errorf("DESIGN.md is missing required architecture marker %q", marker)
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
			t.Errorf("DESIGN.md contains obsolete architecture marker %q", marker)
		}
	}
}
