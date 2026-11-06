package domainmodel

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const ExpectedEndpointCount = 102

var kebabCase = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// ValidateRepository cross-checks manifest references and repository paths.
func ValidateRepository(root string, inventory *EndpointInventory, model *DomainModel, capabilities *CapabilityMap, dependencies *DependencyCapabilities) error {
	if inventory == nil || model == nil || capabilities == nil || dependencies == nil {
		return fmt.Errorf("inventory, model, capability map, and dependency capabilities must all be provided")
	}
	if err := validateEndpointInventory(inventory); err != nil {
		return err
	}
	if err := validateDomainModel(model); err != nil {
		return err
	}
	if err := validateCapabilityMap(capabilities); err != nil {
		return err
	}
	if err := validateDependencyCapabilities(dependencies); err != nil {
		return err
	}
	if err := validateDependencySources(root, dependencies, model); err != nil {
		return err
	}
	if capabilities.Provider != "line" {
		return fmt.Errorf("capability_map.provider %q is unsupported", capabilities.Provider)
	}
	if capabilities.SourceInventory != "docs/line/endpoint-inventory.yaml" {
		return fmt.Errorf("capability_map.source_inventory %q must reference docs/line/endpoint-inventory.yaml", capabilities.SourceInventory)
	}
	if err := requireExistingFile(root, "capability_map.source_inventory", capabilities.SourceInventory); err != nil {
		return err
	}
	for i, subdomain := range model.Subdomains {
		for j, evidence := range subdomain.Evidence {
			if err := requireExistingPath(root, fmt.Sprintf("domain_model.subdomains[%d].evidence[%d]", i, j), evidence); err != nil {
				return err
			}
		}
	}
	contexts := map[string]Context{}
	implementedPackages := map[string]string{}
	for _, context := range model.Contexts {
		contexts[context.ID] = context
	}
	endpoints := map[string]struct{}{}
	for _, endpoint := range inventory.Endpoints {
		endpoints[endpoint.Operation] = struct{}{}
	}
	accounted := map[string]struct{}{}
	for i, operation := range capabilities.Operations {
		path := fmt.Sprintf("capability_map.operations[%d]", i)
		if _, ok := endpoints[operation.Operation]; !ok {
			return fmt.Errorf("%s.operation %q does not exist in endpoint inventory", path, operation.Operation)
		}
		if _, ok := accounted[operation.Operation]; ok {
			return fmt.Errorf("%s.operation %q is duplicated", path, operation.Operation)
		}
		accounted[operation.Operation] = struct{}{}
		if operation.Context != "" {
			if _, ok := contexts[operation.Context]; !ok {
				return fmt.Errorf("%s.context %q does not exist", path, operation.Context)
			}
		}
		if err := requireExistingDir(root, path+".current_implementation", operation.CurrentImplementation); err != nil {
			return err
		}
		if err := requireExistingDir(root, path+".ownership.application", operation.Ownership.Application); err != nil {
			return err
		}
		if err := requireExistingDir(root, path+".ownership.integration", operation.Ownership.Integration); err != nil {
			return err
		}
	}
	if len(accounted) != ExpectedEndpointCount {
		missing := []string{}
		for operation := range endpoints {
			if _, ok := accounted[operation]; !ok {
				missing = append(missing, operation)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("endpoint accounting covers %d of %d operations; missing: %s", len(accounted), ExpectedEndpointCount, strings.Join(missing, ", "))
	}
	for i, context := range model.Contexts {
		path := fmt.Sprintf("domain_model.contexts[%d]", i)
		if context.Package != "" {
			if err := requireExistingDir(root, path+".package", context.Package); err != nil {
				return err
			}
		}
		if context.Status == "implemented" {
			wantPackage := implementedContextPackage(context)
			if context.Package != wantPackage {
				return fmt.Errorf("%s.package %q must equal %q for an implemented context", path, context.Package, wantPackage)
			}
			if previous, exists := implementedPackages[context.Package]; exists {
				return fmt.Errorf("%s.package %q is already owned by context %q", path, context.Package, previous)
			}
			implementedPackages[context.Package] = context.ID
			if err := requireGoBehavior(filepath.Join(root, filepath.FromSlash(context.Package)), path+".package"); err != nil {
				return err
			}
		}
		for j, evidence := range context.Evidence {
			if context.Status == "implemented" && (!strings.HasPrefix(evidence, context.Package+"/") || !strings.HasSuffix(evidence, "_test.go")) {
				return fmt.Errorf("%s.evidence[%d] %q must be a behavior test under %s", path, j, evidence, context.Package)
			}
			if err := requireExistingPath(root, fmt.Sprintf("%s.evidence[%d]", path, j), evidence); err != nil {
				return err
			}
		}
		for j, invariant := range context.Invariants {
			for k, evidence := range invariant.Evidence {
				if err := requireExistingPath(root, fmt.Sprintf("%s.invariants[%d].evidence[%d]", path, j, k), evidence); err != nil {
					return err
				}
			}
		}
		for j, aggregate := range context.Aggregates {
			if aggregate.Repository != "" {
				if err := requireGoSymbol(root, fmt.Sprintf("%s.aggregates[%d].repository", path, j), aggregate.Repository); err != nil {
					return err
				}
			}
			for k, evidence := range aggregate.Evidence {
				if err := requireExistingPath(root, fmt.Sprintf("%s.aggregates[%d].evidence[%d]", path, j, k), evidence); err != nil {
					return err
				}
			}
		}
		for j, ownership := range context.DataOwnership {
			for k, evidence := range ownership.Evidence {
				if err := requireExistingPath(root, fmt.Sprintf("%s.data_ownership[%d].evidence[%d]", path, j, k), evidence); err != nil {
					return err
				}
			}
		}
		for j, adr := range context.ADRs {
			if err := requireExistingFile(root, fmt.Sprintf("%s.adrs[%d]", path, j), adr); err != nil {
				return err
			}
			contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(adr)))
			if err != nil || !strings.Contains(strings.ToLower(string(contents)), context.ID) {
				return fmt.Errorf("%s.adrs[%d] %q does not approve context %q", path, j, adr, context.ID)
			}
		}
		for j, term := range context.Language {
			for k, symbol := range term.CodeSymbols {
				if err := requireGoSymbol(root, fmt.Sprintf("%s.language[%d].code_symbols[%d]", path, j, k), symbol); err != nil {
					return err
				}
			}
		}
	}
	for i, relationship := range model.Relationships {
		if relationship.ADR != "" {
			if err := requireExistingFile(root, fmt.Sprintf("domain_model.relationships[%d].adr", i), relationship.ADR); err != nil {
				return err
			}
		}
	}
	return nil
}

// implementedContextPackage keeps tactical domain models in the Domain layer,
// while read/process models remain application-owned projections and workflows.
func implementedContextPackage(context Context) string {
	if context.ModelKind == "domain-model" {
		return "internal/domain/" + context.ID
	}
	return "internal/application/usecases/" + context.ID
}

func validateEndpointInventory(value *EndpointInventory) error {
	if err := requireSchemaVersion(value.SchemaVersion, 1, "endpoint_inventory"); err != nil {
		return err
	}
	if len(value.Endpoints) != ExpectedEndpointCount {
		return fmt.Errorf("endpoint_inventory.endpoints has %d operations, want %d", len(value.Endpoints), ExpectedEndpointCount)
	}
	for field, text := range map[string]string{
		"source_repository": value.SourceRepository,
		"source_revision":   value.SourceRevision,
		"sdk_module":        value.SDKModule,
		"sdk_version":       value.SDKVersion,
	} {
		if err := requireText("endpoint_inventory."+field, text); err != nil {
			return err
		}
	}
	operations, routes := map[string]struct{}{}, map[string]struct{}{}
	for i, endpoint := range value.Endpoints {
		path := fmt.Sprintf("endpoint_inventory.endpoints[%d]", i)
		for field, text := range map[string]string{"source": endpoint.Source, "group": endpoint.Group, "operation": endpoint.Operation, "method": endpoint.Method, "path": endpoint.Path} {
			if err := requireText(path+"."+field, text); err != nil {
				return err
			}
		}
		if !oneOf(endpoint.Method, "GET", "POST", "PUT", "DELETE", "PATCH") {
			return fmt.Errorf("%s.method %q is invalid", path, endpoint.Method)
		}
		if !strings.HasPrefix(endpoint.Path, "/") {
			return fmt.Errorf("%s.path %q must be absolute", path, endpoint.Path)
		}
		if !oneOf(endpoint.Integration, "not_integrated", "integrated_offline") {
			return fmt.Errorf("%s.integration %q is invalid", path, endpoint.Integration)
		}
		if !oneOf(endpoint.Test, "none", "contract") {
			return fmt.Errorf("%s.test %q is invalid", path, endpoint.Test)
		}
		if !oneOf(endpoint.Access, "review", "required_eligibility_review") {
			return fmt.Errorf("%s.access %q is invalid", path, endpoint.Access)
		}
		if (endpoint.Integration == "integrated_offline") != (endpoint.Test == "contract") {
			return fmt.Errorf("%s integration and test evidence disagree", path)
		}
		if _, ok := operations[endpoint.Operation]; ok {
			return fmt.Errorf("%s.operation %q is duplicated", path, endpoint.Operation)
		}
		operations[endpoint.Operation] = struct{}{}
		route := endpoint.Method + " " + endpoint.Path
		if _, ok := routes[route]; ok {
			return fmt.Errorf("%s route %q is duplicated", path, route)
		}
		routes[route] = struct{}{}
	}
	return nil
}

func validateDomainModel(value *DomainModel) error {
	if err := requireSchemaVersion(value.SchemaVersion, 2, "domain_model"); err != nil {
		return err
	}
	subdomains, contexts := map[string]struct{}{}, map[string]struct{}{}
	for i, subdomain := range value.Subdomains {
		path := fmt.Sprintf("domain_model.subdomains[%d]", i)
		if err := validateID(path+".id", subdomain.ID); err != nil {
			return err
		}
		if _, ok := subdomains[subdomain.ID]; ok {
			return fmt.Errorf("%s.id %q is duplicated", path, subdomain.ID)
		}
		subdomains[subdomain.ID] = struct{}{}
		if !oneOf(subdomain.Classification, "unclassified", "core", "supporting", "generic") {
			return fmt.Errorf("%s.classification %q is invalid", path, subdomain.Classification)
		}
		if err := requireText(path+".purpose", subdomain.Purpose); err != nil {
			return err
		}
		if !oneOf(subdomain.DecisionStatus, "hypothesis", "validated", "implemented") || !oneOf(subdomain.Confidence, "low", "medium", "high") {
			return fmt.Errorf("%s requires valid decision_status and confidence", path)
		}
	}
	for i, context := range value.Contexts {
		path := fmt.Sprintf("domain_model.contexts[%d]", i)
		if err := validateID(path+".id", context.ID); err != nil {
			return err
		}
		if isProviderContext(context.ID) {
			return fmt.Errorf("%s.id %q is a provider or technical boundary", path, context.ID)
		}
		if _, ok := contexts[context.ID]; ok {
			return fmt.Errorf("%s.id %q is duplicated", path, context.ID)
		}
		contexts[context.ID] = struct{}{}
		if _, ok := subdomains[context.Subdomain]; !ok {
			return fmt.Errorf("%s.subdomain %q does not exist", path, context.Subdomain)
		}
		if !oneOf(context.Status, "candidate", "validated", "implemented", "deprecated") {
			return fmt.Errorf("%s.status %q is invalid", path, context.Status)
		}
		if !oneOf(context.Classification, "unclassified", "core", "supporting", "generic") {
			return fmt.Errorf("%s.classification %q is invalid", path, context.Classification)
		}
		if err := requireText(path+".purpose", context.Purpose); err != nil {
			return err
		}
		if !oneOf(context.DecisionStatus, "hypothesis", "validated", "implemented") || !oneOf(context.Confidence, "low", "medium", "high") {
			return fmt.Errorf("%s requires valid decision_status and confidence", path)
		}
		if !oneOf(context.ModelKind, "domain-model", "process-model", "read-model") {
			return fmt.Errorf("%s.model_kind %q is invalid", path, context.ModelKind)
		}
		if len(context.Language) == 0 || len(context.DataOwnership) == 0 || len(context.NonResponsibilities) == 0 {
			return fmt.Errorf("%s requires language, data_ownership, and non_responsibilities", path)
		}
		if context.ModelKind == "domain-model" && len(context.Aggregates) == 0 {
			return fmt.Errorf("%s domain-model requires an aggregate", path)
		}
		if context.ModelKind == "read-model" && len(context.Aggregates) != 0 {
			return fmt.Errorf("%s read-model must not declare aggregates", path)
		}
		if err := validateAggregates(path, context.Aggregates); err != nil {
			return err
		}
		if err := validateDataOwnership(path, context.DataOwnership); err != nil {
			return err
		}
		if err := uniqueTerms(path, context.Language); err != nil {
			return err
		}
		if err := uniqueInvariants(path, context.Invariants); err != nil {
			return err
		}
		if context.Status == "validated" || context.Status == "implemented" {
			if err := requireText(path+".owner", context.Owner); err != nil {
				return err
			}
			if context.Classification == "unclassified" || len(context.NonResponsibilities) == 0 || len(context.Language) == 0 || len(context.Invariants) == 0 || len(context.Capabilities) == 0 || len(context.ADRs) == 0 {
				return fmt.Errorf("%s status %q requires a classification, non_responsibilities, language, invariants, capabilities, and ADR evidence", path, context.Status)
			}
		}
		if context.Status == "implemented" {
			if context.Package == "" {
				return fmt.Errorf("%s.package is required", path)
			}
			if len(context.Evidence) == 0 {
				return fmt.Errorf("%s.evidence is required", path)
			}
		}
	}
	relationships := map[string]struct{}{}
	for i, r := range value.Relationships {
		path := fmt.Sprintf("domain_model.relationships[%d]", i)
		if r.Upstream == r.Downstream {
			return fmt.Errorf("%s must not self-reference", path)
		}
		if _, ok := contexts[r.Upstream]; !ok {
			return fmt.Errorf("%s.upstream %q does not exist", path, r.Upstream)
		}
		if _, ok := contexts[r.Downstream]; !ok {
			return fmt.Errorf("%s.downstream %q does not exist", path, r.Downstream)
		}
		if !oneOf(r.Pattern, "published-language", "anti-corruption-layer", "customer-supplier", "conformist", "shared-kernel") {
			return fmt.Errorf("%s.pattern %q is invalid", path, r.Pattern)
		}
		if !oneOf(r.Interaction, "synchronous", "asynchronous") {
			return fmt.Errorf("%s.interaction %q is invalid", path, r.Interaction)
		}
		if !oneOf(r.DecisionStatus, "hypothesis", "validated", "implemented") || !oneOf(r.Confidence, "low", "medium", "high") {
			return fmt.Errorf("%s requires valid decision_status and confidence", path)
		}
		if r.ConsumerOwnedBy != r.Downstream {
			return fmt.Errorf("%s.consumer_owned_by must equal downstream %q", path, r.Downstream)
		}
		if err := requireText(path+".contract", r.Contract); err != nil {
			return err
		}
		key := r.Upstream + "\x00" + r.Downstream + "\x00" + r.Contract
		if _, exists := relationships[key]; exists {
			return fmt.Errorf("%s duplicates relationship %q", path, r.Upstream+" -> "+r.Downstream+" / "+r.Contract)
		}
		relationships[key] = struct{}{}
		if r.Pattern == "shared-kernel" && r.ADR == "" {
			return fmt.Errorf("%s.adr is required for shared-kernel", path)
		}
	}
	return detectRelationshipCycle(value.Relationships)
}

func validateCapabilityMap(value *CapabilityMap) error {
	if err := requireSchemaVersion(value.SchemaVersion, 1, "capability_map"); err != nil {
		return err
	}
	if err := requireText("capability_map.provider", value.Provider); err != nil {
		return err
	}
	if err := requireText("capability_map.source_inventory", value.SourceInventory); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for i, o := range value.Operations {
		path := fmt.Sprintf("capability_map.operations[%d]", i)
		if err := requireText(path+".operation", o.Operation); err != nil {
			return err
		}
		if _, ok := seen[o.Operation]; ok {
			return fmt.Errorf("%s.operation %q is duplicated", path, o.Operation)
		}
		seen[o.Operation] = struct{}{}
		if !oneOf(o.Status, "mapped", "provisional", "unclassified", "not-domain-relevant") {
			return fmt.Errorf("%s.status %q is invalid", path, o.Status)
		}
		if err := requireText(path+".capability", o.Capability); err != nil {
			return err
		}
		if o.Status == "provisional" && o.Context == "" {
			return fmt.Errorf("%s.context is required for provisional operation", path)
		}
		if oneOf(o.Status, "unclassified", "not-domain-relevant") && o.Context != "" {
			return fmt.Errorf("%s.context must be empty for status %q", path, o.Status)
		}
		if oneOf(o.Status, "unclassified", "not-domain-relevant") {
			if err := requireText(path+".reason", o.Reason); err != nil {
				return err
			}
		}
		if err := requireText(path+".current_implementation", o.CurrentImplementation); err != nil {
			return err
		}
		if err := requireText(path+".ownership.application", o.Ownership.Application); err != nil {
			return err
		}
		if err := requireText(path+".ownership.integration", o.Ownership.Integration); err != nil {
			return err
		}
	}
	return nil
}

func validateDependencyCapabilities(value *DependencyCapabilities) error {
	if err := requireSchemaVersion(value.SchemaVersion, 1, "dependency_capabilities"); err != nil {
		return err
	}
	if value.SourceGoMod != "go.mod" || value.SourceSDKCatalog != "docs/sdk/catalog.toml" {
		return fmt.Errorf("dependency_capabilities sources must be go.mod and docs/sdk/catalog.toml")
	}
	seen := map[string]struct{}{}
	for i, module := range value.Modules {
		path := fmt.Sprintf("dependency_capabilities.modules[%d]", i)
		if err := requireText(path+".module", module.Module); err != nil {
			return err
		}
		if _, ok := seen[module.Module]; ok {
			return fmt.Errorf("%s.module %q is duplicated", path, module.Module)
		}
		seen[module.Module] = struct{}{}
		if err := requireText(path+".version", module.Version); err != nil {
			return err
		}
		if !oneOf(module.CatalogStatus, "declared", "client_ready", "integrated") {
			return fmt.Errorf("%s.catalog_status %q is invalid", path, module.CatalogStatus)
		}
		if !oneOf(module.Classification, "domain-signal", "supporting-capability", "generic-infrastructure", "tooling") {
			return fmt.Errorf("%s.classification %q is invalid", path, module.Classification)
		}
		if len(module.Capabilities) == 0 || len(module.Evidence) == 0 {
			return fmt.Errorf("%s requires capabilities and evidence", path)
		}
		if module.Classification == "generic-infrastructure" && len(module.Contexts) != 0 {
			return fmt.Errorf("%s generic infrastructure must not own contexts", path)
		}
		if err := requireText(path+".assumption", module.Assumption); err != nil {
			return err
		}
	}
	return nil
}

type sdkCatalogManifest struct {
	Modules []struct {
		Module string `toml:"module"`
		Status string `toml:"status"`
	} `toml:"modules"`
}

func validateDependencySources(root string, dependencies *DependencyCapabilities, model *DomainModel) error {
	direct, err := readDirectModules(filepath.Join(root, dependencies.SourceGoMod))
	if err != nil {
		return fmt.Errorf("dependency capabilities: %w", err)
	}
	var catalog sdkCatalogManifest
	if _, err := toml.DecodeFile(filepath.Join(root, dependencies.SourceSDKCatalog), &catalog); err != nil {
		return fmt.Errorf("dependency capabilities: read SDK catalog: %w", err)
	}
	catalogStatus := map[string]string{}
	for _, module := range catalog.Modules {
		catalogStatus[module.Module] = module.Status
	}
	contexts := map[string]struct{}{}
	for _, context := range model.Contexts {
		contexts[context.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	for i, module := range dependencies.Modules {
		path := fmt.Sprintf("dependency_capabilities.modules[%d]", i)
		version, ok := direct[module.Module]
		if !ok {
			return fmt.Errorf("%s.module %q is not a direct go.mod dependency", path, module.Module)
		}
		if version != module.Version {
			return fmt.Errorf("%s.version %q does not match go.mod %q", path, module.Version, version)
		}
		if catalogStatus[module.Module] != module.CatalogStatus {
			return fmt.Errorf("%s.catalog_status %q does not match SDK catalog %q", path, module.CatalogStatus, catalogStatus[module.Module])
		}
		for j, context := range module.Contexts {
			if _, ok := contexts[context]; !ok {
				return fmt.Errorf("%s.contexts[%d] %q does not exist", path, j, context)
			}
		}
		for j, evidence := range module.Evidence {
			if err := requireExistingPath(root, fmt.Sprintf("%s.evidence[%d]", path, j), evidence); err != nil {
				return err
			}
		}
		seen[module.Module] = struct{}{}
	}
	if len(seen) != len(direct) {
		missing := []string{}
		for module := range direct {
			if _, ok := seen[module]; !ok {
				missing = append(missing, module)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("dependency capabilities missing direct go.mod modules: %s", strings.Join(missing, ", "))
	}
	return nil
}

func readDirectModules(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	modules := map[string]string{}
	inBlock := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "require (" {
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
		spec, comment, _ := strings.Cut(line, "//")
		if strings.TrimSpace(comment) == "indirect" {
			continue
		}
		fields := strings.Fields(spec)
		if len(fields) != 2 {
			continue
		}
		modules[fields[0]] = fields[1]
	}
	return modules, scanner.Err()
}

func validateAggregates(path string, values []Aggregate) error {
	seen := map[string]struct{}{}
	for i, aggregate := range values {
		field := fmt.Sprintf("%s.aggregates[%d]", path, i)
		for name, value := range map[string]string{"name": aggregate.Name, "root": aggregate.Root, "consistency_boundary": aggregate.ConsistencyBoundary} {
			if err := requireText(field+"."+name, value); err != nil {
				return err
			}
		}
		if _, ok := seen[aggregate.Name]; ok {
			return fmt.Errorf("%s.name %q is duplicated", field, aggregate.Name)
		}
		seen[aggregate.Name] = struct{}{}
		if len(aggregate.Owns) == 0 || !oneOf(aggregate.Confidence, "low", "medium", "high") {
			return fmt.Errorf("%s requires owns and valid confidence", field)
		}
		if aggregate.Repository != "" && isWorkflowStoreReference(aggregate.Repository) {
			return fmt.Errorf("%s.repository %q must identify an aggregate repository, not a workflow store", field, aggregate.Repository)
		}
	}
	return nil
}

func isWorkflowStoreReference(reference string) bool {
	normalized := strings.ToLower(strings.TrimSpace(reference))
	switch normalized {
	case "internal/application/usecases/messaging-delivery.eventstore",
		"internal/application/usecases/messaging-delivery.deliveryworkflowstore",
		"eventstore",
		"deliveryworkflowstore":
		return true
	default:
		return false
	}
}

func validateDataOwnership(path string, values []DataOwner) error {
	seen := map[string]struct{}{}
	for i, ownership := range values {
		field := fmt.Sprintf("%s.data_ownership[%d]", path, i)
		for name, value := range map[string]string{"resource": ownership.Resource, "owner": ownership.Owner, "write_policy": ownership.WritePolicy} {
			if err := requireText(field+"."+name, value); err != nil {
				return err
			}
		}
		if !oneOf(ownership.Authority, "local-authoritative", "provider-authoritative", "derived-projection") {
			return fmt.Errorf("%s.authority %q is invalid", field, ownership.Authority)
		}
		if _, ok := seen[ownership.Resource]; ok {
			return fmt.Errorf("%s.resource %q is duplicated", field, ownership.Resource)
		}
		seen[ownership.Resource] = struct{}{}
	}
	return nil
}

func validateID(path, id string) error {
	if !kebabCase.MatchString(id) {
		return fmt.Errorf("%s %q must be kebab-case", path, id)
	}
	return nil
}
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func isProviderContext(id string) bool {
	for _, token := range strings.Split(id, "-") {
		if oneOf(token, "line", "google", "firestore", "cloudtasks", "openai", "oauth", "ingestion") {
			return true
		}
	}
	return strings.Contains(id, "cloud-tasks")
}
func uniqueTerms(path string, terms []Term) error {
	seen := map[string]struct{}{}
	for i, t := range terms {
		if err := requireText(fmt.Sprintf("%s.language[%d].term", path, i), t.Term); err != nil {
			return err
		}
		if err := requireText(fmt.Sprintf("%s.language[%d].definition", path, i), t.Definition); err != nil {
			return err
		}
		if _, ok := seen[t.Term]; ok {
			return fmt.Errorf("%s.language[%d].term %q is duplicated", path, i, t.Term)
		}
		seen[t.Term] = struct{}{}
	}
	return nil
}
func uniqueInvariants(path string, values []Invariant) error {
	seen := map[string]struct{}{}
	for i, v := range values {
		if err := requireText(fmt.Sprintf("%s.invariants[%d].id", path, i), v.ID); err != nil {
			return err
		}
		if _, ok := seen[v.ID]; ok {
			return fmt.Errorf("%s.invariants[%d].id %q is duplicated", path, i, v.ID)
		}
		seen[v.ID] = struct{}{}
		if err := requireText(fmt.Sprintf("%s.invariants[%d].statement", path, i), v.Statement); err != nil {
			return err
		}
	}
	return nil
}
func requireExistingPath(root, field, path string) error {
	if filepath.IsAbs(path) || path == "." || filepath.Clean(path) != filepath.FromSlash(path) || strings.Contains(path, "\\") || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %q must be repository-relative", field, path)
	}
	target := filepath.Join(root, filepath.FromSlash(path))
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("%s %q does not exist: %w", field, path, err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return fmt.Errorf("resolve %s %q: %w", field, path, err)
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %q escapes the repository", field, path)
	}
	return nil
}
func requireExistingFile(root, field, path string) error {
	if err := requireExistingPath(root, field, path); err != nil {
		return err
	}
	info, _ := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if info.IsDir() {
		return fmt.Errorf("%s %q is not a file", field, path)
	}
	return nil
}
func requireExistingDir(root, field, path string) error {
	if err := requireText(field, path); err != nil {
		return err
	}
	if err := requireExistingPath(root, field, path); err != nil {
		return err
	}
	info, _ := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if !info.IsDir() {
		return fmt.Errorf("%s %q is not a directory", field, path)
	}
	return nil
}

func requireGoBehavior(root, field string) error {
	var production, tests bool
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			tests = true
		} else {
			production = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: inspect Go behavior: %w", field, err)
	}
	if !production || !tests {
		return fmt.Errorf("%s requires production Go behavior and tests", field)
	}
	return nil
}

func requireGoSymbol(root, field, reference string) error {
	separator := strings.LastIndex(reference, ".")
	if separator <= 0 || separator == len(reference)-1 {
		return fmt.Errorf("%s %q must use package-path.Symbol", field, reference)
	}
	directory, symbol := reference[:separator], reference[separator+1:]
	if err := requireExistingDir(root, field, directory); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(directory)))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(directory), entry.Name()), nil, 0)
		if err != nil {
			return fmt.Errorf("%s: parse %s: %w", field, entry.Name(), err)
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Name.Name == symbol {
					return nil
				}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if spec.Name.Name == symbol {
							return nil
						}
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if name.Name == symbol {
								return nil
							}
						}
					}
				}
			}
		}
	}
	return fmt.Errorf("%s symbol %q does not exist", field, reference)
}
func detectRelationshipCycle(relations []Relationship) error {
	graph := map[string][]string{}
	for _, r := range relations {
		graph[r.Upstream] = append(graph[r.Upstream], r.Downstream)
	}
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(n string) error {
		if state[n] == 1 {
			return fmt.Errorf("domain_model.relationships contains a cycle at %q", n)
		}
		if state[n] == 2 {
			return nil
		}
		state[n] = 1
		for _, next := range graph[n] {
			if err := visit(next); err != nil {
				return err
			}
		}
		state[n] = 2
		return nil
	}
	for n := range graph {
		if err := visit(n); err != nil {
			return err
		}
	}
	return nil
}
