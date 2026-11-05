package main

import (
	"fmt"
	"strings"
)

const agentHandoffContract = "docs/agent-workflows/handoff-contract.md"

var approvedAgentLifecycles = map[string]string{
	"application": "writer",
	"architect":   "read-only-or-writer",
	"domain":      "writer",
	"environment": "read-only-or-writer",
	"integration": "writer",
	"platform":    "read-only-or-writer",
	"researcher":  "read-only",
	"reviewer":    "reviewer",
}

// validateAgentEvidence checks stable lifecycle declarations while leaving the
// rest of the instructions free to describe the role.
func validateAgentEvidence(role string, a agent) error {
	want, ok := approvedAgentLifecycles[role]
	if !ok {
		return fmt.Errorf("agent %s has no approved evidence lifecycle", role)
	}
	instructions := a.DeveloperInstructions
	marker := "Lifecycle: " + want + "."
	if strings.Count(instructions, "Lifecycle:") != 1 || !strings.Contains(instructions, marker) {
		return fmt.Errorf("agent %s must declare exactly one %q evidence lifecycle", role, marker)
	}
	if !strings.Contains(instructions, agentHandoffContract) {
		return fmt.Errorf("agent %s must reference %s", role, agentHandoffContract)
	}
	if strings.Contains(instructions, "same-scope post-change pack") {
		return fmt.Errorf("agent %s contains the obsolete universal Repomix clause", role)
	}

	switch want {
	case "writer":
		if !strings.Contains(instructions, "orchestrator-owned baseline") || !strings.Contains(instructions, "review evidence") {
			return fmt.Errorf("writer agent %s must consume the baseline and provide review evidence after changes", role)
		}
	case "reviewer":
		if !strings.Contains(instructions, "without replacing") || !strings.Contains(instructions, "Not Applicable") {
			return fmt.Errorf("reviewer must validate change pairs and support read-only evidence")
		}
	case "read-only":
		if !strings.Contains(instructions, "Not Applicable") {
			return fmt.Errorf("read-only agent %s must support single-snapshot evidence", role)
		}
	case "read-only-or-writer":
		if !strings.Contains(instructions, "state") && !strings.Contains(instructions, "whether") {
			return fmt.Errorf("hybrid agent %s must report the applied lifecycle", role)
		}
	}
	return nil
}
