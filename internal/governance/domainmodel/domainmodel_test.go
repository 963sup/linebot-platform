package domainmodel

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryManifests(t *testing.T) {
	t.Parallel()
	root := repositoryRoot(t)
	inventory, err := LoadEndpointInventory(filepath.Join(root, "docs/line/endpoint-inventory.yaml"))
	if err != nil {
		t.Fatalf("LoadEndpointInventory(actual) error = %v", err)
	}
	model, err := LoadDomainModel(filepath.Join(root, "docs/domain/model.yaml"))
	if err != nil {
		t.Fatalf("LoadDomainModel(actual) error = %v", err)
	}
	capabilities, err := LoadCapabilityMap(filepath.Join(root, "docs/domain/capability-map.yaml"))
	if err != nil {
		t.Fatalf("LoadCapabilityMap(actual) error = %v", err)
	}
	dependencies, err := LoadDependencyCapabilities(filepath.Join(root, "docs/domain/dependency-capabilities.yaml"))
	if err != nil {
		t.Fatalf("LoadDependencyCapabilities(actual) error = %v", err)
	}
	if err := ValidateRepository(root, inventory, model, capabilities, dependencies); err != nil {
		t.Fatalf("ValidateRepository(actual) error = %v", err)
	}
}

func TestImplementedContextPackageFollowsModelKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind string
		want string
	}{
		{kind: "domain-model", want: "internal/domain/example"},
		{kind: "read-model", want: "internal/application/usecases/example"},
		{kind: "process-model", want: "internal/application/usecases/example"},
	}
	for _, tt := range tests {
		if got := implementedContextPackage(Context{ID: "example", ModelKind: tt.kind}); got != tt.want {
			t.Errorf("implementedContextPackage(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestStrictLoaders(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, kind, yaml, want string }{
		{"domain unknown field", "domain", "schema_version: 2\nsubdomains: []\ncontexts: []\nrelationships: []\nextra: true\n", "field extra not found"},
		{"domain invalid enum", "domain", "schema_version: 2\nsubdomains:\n- id: sample\n  classification: invalid\n  purpose: sample\ncontexts: []\nrelationships: []\n", "classification \"invalid\" is invalid"},
		{"provider context token", "domain", "schema_version: 2\nsubdomains:\n- {id: sample, classification: supporting, decision_status: hypothesis, confidence: low, purpose: sample}\ncontexts:\n- id: google-workspace\n  subdomain: sample\n  status: candidate\n  classification: supporting\n  purpose: sample\n  owner: unassigned\nrelationships: []\n", "provider or technical boundary"},
		{"relationship missing contract", "domain", "schema_version: 2\nsubdomains:\n- {id: sample, classification: supporting, decision_status: hypothesis, confidence: low, purpose: sample}\ncontexts:\n- &ctx {id: first, subdomain: sample, status: candidate, decision_status: hypothesis, confidence: low, model_kind: read-model, classification: supporting, purpose: first, owner: unassigned, non_responsibilities: [none], language: [{term: first-term, definition: first}], data_ownership: [{resource: first-data, authority: derived-projection, owner: first, write_policy: rebuild}]}\n- {id: second, subdomain: sample, status: candidate, decision_status: hypothesis, confidence: low, model_kind: read-model, classification: supporting, purpose: second, owner: unassigned, non_responsibilities: [none], language: [{term: second-term, definition: second}], data_ownership: [{resource: second-data, authority: derived-projection, owner: second, write_policy: rebuild}]}\nrelationships:\n- {upstream: first, downstream: second, pattern: published-language, interaction: synchronous, decision_status: hypothesis, confidence: low, consumer_owned_by: second}\n", "contract must not be empty"},
		{"capability unknown field", "capability", "schema_version: 1\nprovider: line\nsource_inventory: inventory.yaml\noperations: []\nextra: true\n", "field extra not found"},
		{"capability duplicate operation", "capability", "schema_version: 1\nprovider: line\nsource_inventory: inventory.yaml\noperations:\n- &op\n  operation: op\n  status: mapped\n  capability: sample\n  current_implementation: internal/application\n  ownership: {application: internal/application, integration: internal/adapter/outbound/line}\n- *op\n", "operation \"op\" is duplicated"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := writeFixture(t, tt.yaml)
			var err error
			if tt.kind == "domain" {
				_, err = LoadDomainModel(path)
			} else {
				_, err = LoadCapabilityMap(path)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("loader(%s) error = %v, want containing %q", tt.name, err, tt.want)
			}
		})
	}
}

func TestLoadEndpointInventory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{"valid", func(s string) string { return s }, ""},
		{"unknown field", func(s string) string { return s + "unknown: true\n" }, "field unknown not found"},
		{"wrong count", func(s string) string {
			return strings.Replace(s, "\n  - {source: source, group: group, operation: op101", "\n  # removed {source: source, group: group, operation: op101", 1)
		}, "has 101 operations, want 102"},
		{"duplicate operation", func(s string) string { return strings.Replace(s, "operation: op101", "operation: op100", 1) }, "operation \"op100\" is duplicated"},
		{"invalid method", func(s string) string { return strings.Replace(s, "method: GET", "method: CONNECT", 1) }, "method \"CONNECT\" is invalid"},
		{"contract mismatch", func(s string) string { return strings.Replace(s, "test: contract", "test: none", 1) }, "integration and test evidence disagree"},
	}
	base := inventoryYAML()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := LoadEndpointInventory(writeFixture(t, tt.mutate(base)))
			if tt.want == "" && err != nil {
				t.Errorf("LoadEndpointInventory(valid) error = %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Errorf("LoadEndpointInventory(%s) error = %v, want containing %q", tt.name, err, tt.want)
			}
		})
	}
}

func TestValidateRepositoryFailures(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name, want string
		mutate     func(*DomainModel, *CapabilityMap)
	}{
		{"dangling context", "does not exist", func(_ *DomainModel, c *CapabilityMap) {
			for i := range c.Operations {
				if c.Operations[i].Status == "provisional" {
					c.Operations[i].Context = "missing-context"
					return
				}
			}
		}},
		{"missing implementation", "does not exist", func(_ *DomainModel, c *CapabilityMap) { c.Operations[0].CurrentImplementation = "internal/missing" }},
		{"implementation path escapes repository", "repository-relative", func(_ *DomainModel, c *CapabilityMap) { c.Operations[0].CurrentImplementation = "../outside" }},
		{"missing evidence", "does not exist", func(m *DomainModel, _ *CapabilityMap) {
			m.Contexts[0].Evidence = []string{m.Contexts[0].Package + "/missing_test.go"}
		}},
		{"missing subdomain evidence", "domain_model.subdomains[0].evidence[0]", func(m *DomainModel, _ *CapabilityMap) {
			m.Subdomains[0].Evidence = []string{"internal/missing-subdomain-evidence"}
		}},
		{"missing aggregate evidence", "aggregates[0].evidence[0]", func(m *DomainModel, _ *CapabilityMap) {
			m.Contexts[0].Aggregates[0].Evidence = []string{"internal/missing-aggregate-evidence"}
		}},
		{"missing aggregate repository symbol", "aggregates[0].repository", func(m *DomainModel, _ *CapabilityMap) {
			m.Contexts[0].Aggregates[0].Repository += "Missing"
		}},
		{"missing data ownership evidence", "data_ownership[0].evidence[0]", func(m *DomainModel, _ *CapabilityMap) {
			m.Contexts[0].DataOwnership[0].Evidence = []string{"internal/missing-data-ownership-evidence"}
		}},
		{"implemented context missing ADR", "ADR evidence", func(m *DomainModel, _ *CapabilityMap) { m.Contexts[0].Status = "implemented"; m.Contexts[0].ADRs = nil }},
		{"implemented context wrong package", "must equal", func(m *DomainModel, _ *CapabilityMap) {
			m.Contexts[0].Status = "implemented"
			m.Contexts[0].ADRs = []string{"docs/architecture/decisions/0003-tactical-models-without-strategic-promotion.md"}
			m.Contexts[0].Package = "internal/domain"
		}},
		{"implemented evidence outside package", "behavior test under", func(m *DomainModel, _ *CapabilityMap) {
			m.Contexts[0].Status = "implemented"
			m.Contexts[0].ADRs = []string{"docs/architecture/decisions/0003-tactical-models-without-strategic-promotion.md"}
			m.Contexts[0].Evidence = []string{"tests/architecture/dependencies_test.go"}
		}},
		{"missing code symbol", "does not exist", func(m *DomainModel, _ *CapabilityMap) {
			m.Contexts[0].Language[0].CodeSymbols = []string{"internal/domain/messaging-delivery/aggregates.Missing"}
		}},
		{"incomplete accounting", "covers 101 of 102", func(_ *DomainModel, c *CapabilityMap) { c.Operations = c.Operations[:101] }},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inventory, model, capabilities, dependencies := loadActual(t, root)
			tt.mutate(model, capabilities)
			err := ValidateRepository(root, inventory, model, capabilities, dependencies)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("ValidateRepository(%s) error = %v, want containing %q", tt.name, err, tt.want)
			}
		})
	}
}

func TestUnsupportedEndpointCannotClaimIntegration(t *testing.T) {
	root := repositoryRoot(t)
	inventory, model, capabilities, dependencies := loadActual(t, root)
	for i := range inventory.Endpoints {
		if inventory.Endpoints[i].Operation == "leaveRoom" {
			inventory.Endpoints[i].Integration = "integrated_offline"
			inventory.Endpoints[i].Test = "contract"
		}
	}
	err := ValidateRepository(root, inventory, model, capabilities, dependencies)
	if err == nil || !strings.Contains(err.Error(), "integrated endpoint requires an implementation") {
		t.Fatalf("unsupported operation claimed implementation: %v", err)
	}
}

func TestDependencyCapabilityFailures(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name, want string
		mutate     func(*DependencyCapabilities)
	}{
		{"version drift", "does not match go.mod", func(d *DependencyCapabilities) { d.Modules[0].Version = "v0.0.0" }},
		{"catalog status drift", "does not match SDK catalog", func(d *DependencyCapabilities) { d.Modules[0].CatalogStatus = "declared" }},
		{"generic owns context", "generic infrastructure must not own contexts", func(d *DependencyCapabilities) { d.Modules[0].Contexts = []string{"messaging-delivery"} }},
		{"missing direct module", "missing direct go.mod modules", func(d *DependencyCapabilities) { d.Modules = d.Modules[1:] }},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inventory, model, capabilities, dependencies := loadActual(t, root)
			tt.mutate(dependencies)
			err := ValidateRepository(root, inventory, model, capabilities, dependencies)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("ValidateRepository(%s) error = %v, want containing %q", tt.name, err, tt.want)
			}
		})
	}
}

func TestModelKindRules(t *testing.T) {
	root := repositoryRoot(t)
	_, model, _, _ := loadActual(t, root)
	for i := range model.Contexts {
		if model.Contexts[i].ID == "membership" {
			model.Contexts[i].Aggregates = []Aggregate{{Name: "Membership", Root: "membership", ConsistencyBoundary: "invented", Owns: []string{"state"}, Confidence: "low"}}
			break
		}
	}
	if err := validateDomainModel(model); err == nil || !strings.Contains(err.Error(), "read-model must not declare aggregates") {
		t.Fatalf("read-model aggregate error = %v", err)
	}
}

func TestAggregateRepositoryRules(t *testing.T) {
	root := repositoryRoot(t)
	_, model, _, _ := loadActual(t, root)

	var candidate Aggregate
	for _, context := range model.Contexts {
		if context.ID == "messaging-delivery" {
			candidate = context.Aggregates[0]
			break
		}
	}
	if candidate.Name == "" {
		t.Fatal("messaging-delivery aggregate fixture is missing")
	}

	tests := []struct {
		name       string
		status     string
		repository string
		want       string
	}{
		{name: "candidate may omit repository", status: "candidate"},
		{name: "implemented may use a capability workflow store without an aggregate repository", status: "implemented"},
		{name: "implemented accepts aggregate repository", status: "implemented", repository: "internal/domain/messaging-delivery/repositories.DeliveryRepository"},
		{name: "implemented accepts event sourced aggregate repository", status: "implemented", repository: "internal/domain/orders/repositories.OrderEventStore"},
		{name: "event store is a workflow alias", status: "candidate", repository: "internal/application/usecases/messaging-delivery.EventStore", want: "not a workflow store"},
		{name: "workflow store is not an aggregate repository", status: "implemented", repository: "DeliveryWorkflowStore", want: "not a workflow store"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := candidate
			value.Repository = tt.repository
			err := validateAggregates("domain_model.contexts[fixture]", []Aggregate{value})
			if tt.want == "" && err != nil {
				t.Fatalf("validateAggregates() error = %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("validateAggregates() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRequireExistingPathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "evidence.go"), []byte("package evidence\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := requireExistingPath(root, "evidence", "escape/evidence.go"); err == nil || !strings.Contains(err.Error(), "escapes the repository") {
		t.Fatalf("symlink escape error = %v", err)
	}
}

func TestRenderContextMapDeterministic(t *testing.T) {
	t.Parallel()
	root := repositoryRoot(t)
	_, model, capabilities, dependencies := loadActual(t, root)
	first, err := RenderContextMap(model, capabilities, dependencies)
	if err != nil {
		t.Fatalf("RenderContextMap(first) error = %v", err)
	}
	for left, right := 0, len(model.Contexts)-1; left < right; left, right = left+1, right-1 {
		model.Contexts[left], model.Contexts[right] = model.Contexts[right], model.Contexts[left]
	}
	for left, right := 0, len(capabilities.Operations)-1; left < right; left, right = left+1, right-1 {
		capabilities.Operations[left], capabilities.Operations[right] = capabilities.Operations[right], capabilities.Operations[left]
	}
	second, err := RenderContextMap(model, capabilities, dependencies)
	if err != nil {
		t.Fatalf("RenderContextMap(reordered) error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("RenderContextMap(reordered) differs from original")
	}
	for _, want := range []string{"```mermaid", "Subdomains", "Aggregates and Data Ownership", "MSG-001", "messaging-delivery", "Endpoint Capabilities", "Go Module Capability Evidence", "| Decision | Confidence |", "RecipientSelection / synchronous / published-language / hypothesis / medium"} {
		if !bytes.Contains(first, []byte(want)) {
			t.Errorf("RenderContextMap output missing %q", want)
		}
	}
}

func loadActual(t *testing.T, root string) (*EndpointInventory, *DomainModel, *CapabilityMap, *DependencyCapabilities) {
	t.Helper()
	i, err := LoadEndpointInventory(filepath.Join(root, "docs/line/endpoint-inventory.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadDomainModel(filepath.Join(root, "docs/domain/model.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := LoadCapabilityMap(filepath.Join(root, "docs/domain/capability-map.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := LoadDependencyCapabilities(filepath.Join(root, "docs/domain/dependency-capabilities.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return i, m, c, d
}
func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("filepath.Abs repository root error = %v", err)
	}
	return root
}
func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile fixture error = %v", err)
	}
	return path
}
func inventoryYAML() string {
	var b strings.Builder
	b.WriteString("schema_version: 1\nsource_repository: repo\nsource_revision: revision\nsdk_module: module\nsdk_version: version\nlive_channel_verified: false\nendpoints:\n")
	for i := 0; i < ExpectedEndpointCount; i++ {
		fmt.Fprintf(&b, "  - {source: source, group: group, operation: op%d, method: GET, path: /path/%d, integration: integrated_offline, test: contract, access: review}\n", i, i)
	}
	return b.String()
}
