package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// Extractor connects to a running New API instance, reads the existing api.json
// skeleton, and collects real response data from every reachable endpoint.
type Extractor struct {
	cfg   ExtractorConfig
	httpc *http.Client
}

// NewExtractor creates a new Extractor with the given config.
func NewExtractor(cfg ExtractorConfig) *Extractor {
	return &Extractor{
		cfg: cfg,
		httpc: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Extract reads the skeleton OpenAPI spec at skeletonPath, sends a GET request
// to each endpoint in the running New API instance, infers response schemas,
// and returns the collected results.
func (e *Extractor) Extract(ctx context.Context, skeletonPath string) (*ExtractionResult, error) {
	skeletons, err := e.loadSkeleton(skeletonPath)
	if err != nil {
		return nil, fmt.Errorf("load skeleton: %w", err)
	}

	result := &ExtractionResult{
		Components: make(map[string]*SchemaField),
	}

	for _, skel := range skeletons {
		// Only extract from GET endpoints — we never mutate the upstream API.
		if skel.Method != "GET" {
			continue
		}

		data, err := e.fetchEndpoint(ctx, skel.Method, skel.Path)
		if err != nil {
			// Log the error but continue with other endpoints.
			result.Endpoints = append(result.Endpoints, EndpointSchema{
				Endpoint: EndpointInfo{
					Path:        skel.Path,
					Method:      skel.Method,
					OperationID: skel.OperationID,
					Tags:        skel.Tags,
				},
			})
			continue
		}

		// Parse the response body as generic JSON.
		var parsed any
		if err := json.Unmarshal(data, &parsed); err != nil {
			// Non-JSON response; skip.
			continue
		}

		schema := InferSchema(parsed)
		if schema == nil {
			schema = &SchemaField{Type: "object"}
		}

		// Collect named component schemas from the inferred schema.
		e.collectComponents(schema, result.Components)

		result.Endpoints = append(result.Endpoints, EndpointSchema{
			Endpoint: EndpointInfo{
				Path:        skel.Path,
				Method:      skel.Method,
				OperationID: skel.OperationID,
				Tags:        skel.Tags,
			},
			ResponseBody: schema,
		})
	}

	return result, nil
}

// loadSkeleton reads an OpenAPI 3.0 spec and extracts the endpoint metadata
// (path, method, operationId, tags) without requiring response schemas.
func (e *Extractor) loadSkeleton(path string) ([]SkeletonEndpoint, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skeleton file %s: %w", path, err)
	}

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(raw)
	if err != nil {
		return nil, fmt.Errorf("parse skeleton spec: %w", err)
	}

	var endpoints []SkeletonEndpoint
	for pathStr, pathItem := range doc.Paths.Map() {
		for method, op := range pathItem.Operations() {
			ep := SkeletonEndpoint{
				Method:      strings.ToUpper(method),
				OperationID: op.OperationID,
				Tags:        op.Tags,
				Summary:     op.Summary,
			}
			// clean duplicated path
			ep.Path = pathStr
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints, nil
}

// fetchEndpoint sends a GET request to the upstream New API endpoint and returns
// the raw response body. On non-2xx responses an error is returned with a
// summary of the response body for diagnostics.
func (e *Extractor) fetchEndpoint(ctx context.Context, method, path string) ([]byte, error) {
	url := e.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	e.addAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		summary := string(body)
		if len(summary) > 200 {
			summary = summary[:200]
		}
		return nil, fmt.Errorf("%s %s: HTTP %d — %s", method, path, resp.StatusCode, summary)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

// collectComponents walks an inferred schema tree and registers named objects
// as component schemas for later use in $ref references.
func (e *Extractor) collectComponents(schema *SchemaField, components map[string]*SchemaField) {
	if schema == nil {
		return
	}

	// Register named objects as components.
	if schema.Type == "object" && schema.Name != "" {
		if _, exists := components[schema.Name]; !exists {
			components[schema.Name] = schema
		}
	}

	// Recurse into properties.
	for i := range schema.Properties {
		e.collectComponents(&schema.Properties[i], components)
	}

	// Recurse into array items.
	if schema.Items != nil {
		e.collectComponents(schema.Items, components)
	}
}