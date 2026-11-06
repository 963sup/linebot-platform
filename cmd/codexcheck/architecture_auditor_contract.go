package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type architectureAuditorClause struct {
	name string
	text string
}

var architectureAuditorInstructionClauses = []architectureAuditorClause{
	{name: "complete Repomix production-Go snapshot", text: "Repomix MCP pack_codebase includePatterns **/*.go, ignorePatterns **/*_test.go"},
	{name: "read before edit", text: "Do not edit while reading or analyzing"},
	{name: "gopls confirmation", text: "Confirm current symbols and dependencies through gopls MCP where available"},
	{name: "six-part decision matrix", text: "mandatory six-part decision matrix"},
	{name: "Domain classification", text: "Aggregate, Aggregate Root, Entity, Value Object, Domain Service, Policy, Specification, Factory, Repository Interface, Domain Event, Domain Error"},
	{name: "Application classification", text: "Use Case, Application Service, Command, Query, DTO, Input Port, Output Port, Unit of Work, Transaction Boundary, Event Handler, Authorization"},
	{name: "non-scaffolding rule", text: "never as a scaffold checklist"},
	{name: "abstraction evidence", text: "Require evidence of a business invariant, owner, production consumer, independent behavior, and acceptance test before recommending a new abstraction"},
	{name: "unused component removal", text: "Recommend removal when a Factory, Repository, Domain Event, wrapper, mapper, or other abstraction has no production consumer or necessary behavior"},
	{name: "composition-root ownership", text: "composition roots stay in cmd/* and internal/bootstrap"},
	{name: "capability-local Application ports", text: "Application capability ports stay with their consuming capability rather than a global ports package"},
	{name: "repository and workflow-store ownership", text: "Domain repositories express Aggregate persistence while workflow stores remain Application-owned"},
	{name: "repository Adapter boundary", text: "Repository Adapters live under outbound adapters while technical clients/config/lifecycle live in Infrastructure"},
	{name: "authentication and authorization separation", text: "authentication integration remains distinct from Application authorization"},
	{name: "Infrastructure ownership limit", text: "Infrastructure must not absorb Bootstrap, Application, Adapter, or Domain responsibilities"},
	{name: "authority and visual correction", text: "correct a view that conflicts with the Blueprint, Domain manifests, accepted ADRs, executable contracts, or verified behavior instead of distorting code to match it"},
	{name: "candidate Context evidence", text: "Candidate Contexts are not promoted by diagrams, packages, SDKs, or tests alone"},
	{name: "semantic derivation order", text: "Business Fact -> Invariant -> Owner -> Boundary -> Component Kind -> Name -> Acceptance Test"},
	{name: "structure derivation", text: "Structure = Facts + Invariants + Ownership + Dependencies"},
	{name: "abstraction derivation", text: "Abstraction = Necessary Behavior + Real Boundary + Production Consumer"},
	{name: "naming derivation", text: "Name = shortest unambiguous official term within its Context"},
	{name: "Semantic Registry authority", text: "docs/domain/model.yaml term, definition, forbidden_aliases, and code_symbols as the Semantic Registry"},
	{name: "unsupported generic abstraction", text: "A Manager, Service, or Interface without necessary behavior and a production consumer is invalid"},
	{name: "consumer-named interface", text: "Name an interface for its consuming capability, not its provider or implementation"},
	{name: "non-stuttering symbol", text: "Remove package and receiver context repeated in a symbol"},
	{name: "Domain Event semantics", text: "Name Domain Events as past facts and require a production consumer"},
	{name: "forbidden alias", text: "Never recommend a forbidden_alias as the official name"},
	{name: "MCP evidence roles", text: "Use MCPs by evidence role, not because they are configured"},
	{name: "Repomix semantic inventory", text: "Repomix MCP provides the complete production-Go inventory and excludes *_test.go"},
	{name: "gopls semantic confirmation", text: "gopls MCP confirms current symbols, references, implementations, rename impact, and diagnostics"},
	{name: "Context7 scope", text: "Context7 is only for version-specific Go or SDK claims"},
	{name: "GitHub scope", text: "GitHub MCP is only for remote PR and CI evidence"},
	{name: "provider MCP scope", text: "provider, cloud, Terraform, LINE, Google, and Firestore tools are used only when their provider semantics are in scope"},
	{name: "semantic finding schema", text: "official term, owner, component kind, current name, mismatch, smallest correction, production consumer, and acceptance test"},
	{name: "existing-code full read", text: "For existing-code convergence, read all production Go before deciding and inventory every architecture-bearing package-level declaration"},
	{name: "core semantic mapping", text: "Map Domain and Application packages, exported types, interfaces, functions, and methods to the Semantic Registry or a named capability owner"},
	{name: "technical naming boundary", text: "Check Adapter and Infrastructure names against their provider or technical responsibility without promoting them to Domain terms"},
	{name: "local naming boundary", text: "local variables remain outside the registry and follow contextual Go naming"},
	{name: "existing-code decisions", text: "Classify every item as Keep, Rename, Merge, Delete, or Exception"},
	{name: "no unmapped core", text: "leave no unresolved or unmapped core item"},
	{name: "exception evidence", text: "An Exception requires an owner, compatibility reason, exit condition, and acceptance test, and a persistent exception requires an accepted ADR"},
	{name: "existing-code finding schema", text: "current symbol, layer/context, official term, decision, evidence, target symbol or removal, production consumer, compatibility impact, and acceptance test"},
	{name: "existing-code migration order", text: "Semantic Registry and contracts; types, interfaces, and package-level declarations; call sites and composition; tests; Markdown and architecture assets; same-pattern Repomix review; independent review"},
	{name: "existing-code plan-before-write", text: "Do not implement while inventorying or planning"},
	{name: "finding schema", text: "layer/component, business invariant, current evidence, mismatch, root cause, owner, smallest correction, compatibility impact, acceptance tests, and residual risk"},
	{name: "read-only lifecycle", text: "Lifecycle: read-only."},
	{name: "no autonomous delegation or writes", text: "do not spawn subagents, write files, commit, push"},
}

var architectureAuditorWorkflowClauses = []architectureAuditorClause{
	{name: "production-Go inclusion", text: "includePatterns: **/*.go"},
	{name: "test exclusion", text: "ignorePatterns: **/*_test.go"},
	{name: "complete-read gate", text: "Read every packed production Go file in bounded contiguous ranges"},
	{name: "no edit before analysis", text: "No code or architecture asset edits before the read and analysis gates complete"},
	{name: "six-part decision matrix", text: "Mandatory architecture decision matrix"},
	{name: "Domain classification", text: "Aggregate, Aggregate Root, Entity, Value Object, Domain Service, Policy, Specification, Factory, Repository Interface, Domain Event, Domain Error"},
	{name: "Application classification", text: "Use Case, Application Service, Command, Query, input/output DTO, Input Port, Output Port, Unit of Work, Transaction Boundary, Event Handler, Authorization"},
	{name: "Adapter classification", text: "Repository, API client and event publisher/consumer adapters"},
	{name: "Infrastructure and Bootstrap classification", text: "Infrastructure and Bootstrap"},
	{name: "Strategic DDD classification", text: "Domain, Subdomain, Bounded Context, Ubiquitous Language, Context Map, Integration Contract, Anti-Corruption Layer"},
	{name: "global constraints", text: "Dependency Rule, high cohesion, low coupling, encapsulation, explicit ownership, Go idioms, KISS, YAGNI, DRY"},
	{name: "non-scaffolding rule", text: "not a scaffold checklist"},
	{name: "unused component removal", text: "An unused Factory, Repository or Domain Event with no production consumer or necessary behavior must be proposed for removal; do not add components to complete a pattern"},
	{name: "capability-local Application ports", text: "never propose a global Application ports bucket"},
	{name: "repository and workflow-store ownership", text: "A Domain repository expresses Aggregate persistence semantics. An Application workflow store or external capability remains owned by its consuming capability"},
	{name: "repository Adapter boundary", text: "Repository implementations are outbound adapters; database connections, SDK clients, configuration and lifecycle are Infrastructure concerns"},
	{name: "authentication integration ownership", text: "Provider authentication and token verification are integration boundaries"},
	{name: "Application authorization ownership", text: "Use-case authorization remains an Application decision expressed in business terms"},
	{name: "Infrastructure ownership limit", text: "internal/infrastructure` must not absorb Bootstrap, Application, Adapter or Domain responsibilities"},
	{name: "candidate Context evidence", text: "Candidate Contexts are not promoted by packages, diagrams, SDKs or tests alone"},
	{name: "authority order", text: "Authority order is: `docs/architecture/blueprint.md` and architecture tests"},
	{name: "visual correction", text: "Correct the lower-authority source when evidence conflicts"},
	{name: "finding schema", text: "layer/component, business invariant, current evidence, mismatch, root cause, owner, smallest correction, compatibility impact, acceptance tests and residual risk"},
	{name: "not-applicable outcome", text: "Explicitly report `Not Applicable` when a building block is unnecessary"},
}

func validateArchitectureAuditorContractText(kind, text string, clauses []architectureAuditorClause) error {
	for _, clause := range clauses {
		if !strings.Contains(text, clause.text) {
			return fmt.Errorf("architecture_auditor %s contract missing %s clause %q", kind, clause.name, clause.text)
		}
	}
	return nil
}

func validateArchitectureAuditorInstructions(instructions string) error {
	return validateArchitectureAuditorContractText("instructions", instructions, architectureAuditorInstructionClauses)
}

func validateArchitectureAuditorWorkflow(root string) error {
	path := filepath.Join(root, "docs", "agent-workflows", "go-architecture-audit.md")
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read architecture_auditor workflow: %w", err)
	}
	return validateArchitectureAuditorContractText("workflow", string(contents), architectureAuditorWorkflowClauses)
}
