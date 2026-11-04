package outbound

// LINEObject is a transport-neutral structured value used at the application
// boundary for LINE operations whose provider schema is intentionally open.
// JSON decoding and encoding belong to inbound and outbound adapters.
type LINEObject map[string]any

// LINEArray is the array counterpart of LINEObject.
type LINEArray []any

// LINEResult is a typed application result. It deliberately has no JSON tags
// and contains no LINE SDK types.
type LINEResult struct {
	Value any
}

// LINERequest is a typed application command payload. Operation-specific port
// methods retain the semantic operation name while adapters own wire mapping.
type LINERequest struct {
	Value any
}
