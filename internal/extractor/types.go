// Package extractor provides tools to extract accurate OpenAPI schemas from a
// running New API instance. It sends read-only GET requests to each endpoint
// listed in the existing api.json skeleton, infers response schemas from real
// JSON payloads, and merges them back into a complete OpenAPI 3.0.1 document.
package extractor

// ExtractorConfig configures the extractor's connection to a running New API instance.
type ExtractorConfig struct {
	// BaseURL is the New API base address, e.g. "http://localhost:3000".
	BaseURL string

	// SystemKey is the admin access_token sent as Bearer in the Authorization header.
	SystemKey string

	// UserID is the value for the New-Api-User header (required for admin endpoints).
	UserID string
}

// EndpointInfo holds metadata about a single API endpoint read from the skeleton spec.
type EndpointInfo struct {
	Path        string
	Method      string
	OperationID string
	Tags        []string
}

// SchemaField represents an inferred OpenAPI schema for a JSON value.
type SchemaField struct {
	Name        string        `json:"name,omitempty"`
	Type        string        `json:"type,omitempty"`       // string, integer, number, boolean, array, object
	Format      string        `json:"format,omitempty"`     // date-time, int64, float, etc.
	Nullable    bool          `json:"nullable,omitempty"`   // true when the field was null or zero-value
	Description string        `json:"description,omitempty"`
	Enum        []any         `json:"enum,omitempty"`       // inferred from repeated values
	Properties  []SchemaField `json:"properties,omitempty"` // nested object fields
	Items       *SchemaField  `json:"items,omitempty"`      // array element type
	Ref         string        `json:"$ref,omitempty"`       // "#/components/schemas/..." reference
}

// ExtractionResult holds all data collected by the extractor.
type ExtractionResult struct {
	Endpoints  []EndpointSchema
	Components map[string]*SchemaField // named schemas for components/schemas/
}

// EndpointSchema pairs an endpoint with its inferred response schema.
type EndpointSchema struct {
	Endpoint     EndpointInfo
	ResponseBody *SchemaField
}

// SkeletonEndpoint is the reduced representation of an endpoint from the skeleton spec.
type SkeletonEndpoint struct {
	Path        string
	Method      string
	OperationID string
	Tags        []string
	Summary     string
}