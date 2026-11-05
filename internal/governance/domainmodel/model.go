// Package domainmodel loads and validates the repository's strategic DDD manifests.
package domainmodel

// EndpointInventory is the authoritative snapshot of provider operations.
type EndpointInventory struct {
	SchemaVersion       int        `yaml:"schema_version"`
	SourceRepository    string     `yaml:"source_repository"`
	SourceRevision      string     `yaml:"source_revision"`
	SDKModule           string     `yaml:"sdk_module"`
	SDKVersion          string     `yaml:"sdk_version"`
	LiveChannelVerified bool       `yaml:"live_channel_verified"`
	Endpoints           []Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Source      string `yaml:"source"`
	Group       string `yaml:"group"`
	Operation   string `yaml:"operation"`
	Method      string `yaml:"method"`
	Path        string `yaml:"path"`
	Integration string `yaml:"integration"`
	Test        string `yaml:"test"`
	Access      string `yaml:"access"`
}

type DomainModel struct {
	SchemaVersion int            `yaml:"schema_version"`
	Subdomains    []Subdomain    `yaml:"subdomains"`
	Contexts      []Context      `yaml:"contexts"`
	Relationships []Relationship `yaml:"relationships"`
}

type Subdomain struct {
	ID             string   `yaml:"id"`
	Classification string   `yaml:"classification"`
	DecisionStatus string   `yaml:"decision_status"`
	Confidence     string   `yaml:"confidence"`
	Purpose        string   `yaml:"purpose"`
	Evidence       []string `yaml:"evidence"`
}

type Context struct {
	ID                  string      `yaml:"id"`
	Subdomain           string      `yaml:"subdomain"`
	Status              string      `yaml:"status"`
	DecisionStatus      string      `yaml:"decision_status"`
	Confidence          string      `yaml:"confidence"`
	ModelKind           string      `yaml:"model_kind"`
	Classification      string      `yaml:"classification"`
	Purpose             string      `yaml:"purpose"`
	Owner               string      `yaml:"owner"`
	Package             string      `yaml:"package"`
	NonResponsibilities []string    `yaml:"non_responsibilities"`
	Language            []Term      `yaml:"language"`
	Owns                []string    `yaml:"owns"`
	Invariants          []Invariant `yaml:"invariants"`
	Aggregates          []Aggregate `yaml:"aggregates"`
	DataOwnership       []DataOwner `yaml:"data_ownership"`
	Capabilities        []string    `yaml:"capabilities"`
	ADRs                []string    `yaml:"adrs"`
	OpenQuestions       []string    `yaml:"open_questions"`
	Evidence            []string    `yaml:"evidence"`
}

type Aggregate struct {
	Name                string   `yaml:"name"`
	Root                string   `yaml:"root"`
	ConsistencyBoundary string   `yaml:"consistency_boundary"`
	Owns                []string `yaml:"owns"`
	Repository          string   `yaml:"repository"`
	Confidence          string   `yaml:"confidence"`
	Evidence            []string `yaml:"evidence"`
}

type DataOwner struct {
	Resource    string   `yaml:"resource"`
	Authority   string   `yaml:"authority"`
	Owner       string   `yaml:"owner"`
	WritePolicy string   `yaml:"write_policy"`
	Evidence    []string `yaml:"evidence"`
}

type Term struct {
	Term             string   `yaml:"term"`
	Definition       string   `yaml:"definition"`
	ForbiddenAliases []string `yaml:"forbidden_aliases"`
	CodeSymbols      []string `yaml:"code_symbols"`
}

type Invariant struct {
	ID        string   `yaml:"id"`
	Statement string   `yaml:"statement"`
	Evidence  []string `yaml:"evidence"`
}

type Relationship struct {
	Upstream        string `yaml:"upstream"`
	Downstream      string `yaml:"downstream"`
	Pattern         string `yaml:"pattern"`
	Contract        string `yaml:"contract"`
	Interaction     string `yaml:"interaction"`
	DecisionStatus  string `yaml:"decision_status"`
	Confidence      string `yaml:"confidence"`
	ConsumerOwnedBy string `yaml:"consumer_owned_by"`
	ADR             string `yaml:"adr"`
}

type DependencyCapabilities struct {
	SchemaVersion    int                    `yaml:"schema_version"`
	SourceGoMod      string                 `yaml:"source_go_mod"`
	SourceSDKCatalog string                 `yaml:"source_sdk_catalog"`
	Modules          []DependencyCapability `yaml:"modules"`
}

type DependencyCapability struct {
	Module         string   `yaml:"module"`
	Version        string   `yaml:"version"`
	CatalogStatus  string   `yaml:"catalog_status"`
	Classification string   `yaml:"classification"`
	Capabilities   []string `yaml:"capabilities"`
	Contexts       []string `yaml:"contexts"`
	Evidence       []string `yaml:"evidence"`
	Assumption     string   `yaml:"assumption"`
}

type CapabilityMap struct {
	SchemaVersion   int                   `yaml:"schema_version"`
	Provider        string                `yaml:"provider"`
	SourceInventory string                `yaml:"source_inventory"`
	Operations      []CapabilityOperation `yaml:"operations"`
}

type CapabilityOperation struct {
	Operation             string    `yaml:"operation"`
	Status                string    `yaml:"status"`
	Capability            string    `yaml:"capability"`
	Context               string    `yaml:"context"`
	CurrentImplementation string    `yaml:"current_implementation"`
	Ownership             Ownership `yaml:"ownership"`
	Reason                string    `yaml:"reason"`
}

type Ownership struct {
	Application string `yaml:"application"`
	Integration string `yaml:"integration"`
}
