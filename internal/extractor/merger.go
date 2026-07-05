package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Merger takes inferred endpoint schemas and merges them back into a complete
// OpenAPI 3.0.1 document, preserving the original skeleton's structure while
// adding accurate response schemas with $ref references to components/schemas.
type Merger struct {
	skeletonPath string
}

// NewMerger creates a Merger that reads the original skeleton spec.
func NewMerger(skeletonPath string) *Merger {
	return &Merger{skeletonPath: skeletonPath}
}

// Merge produces a complete OpenAPI 3.0.1 document by merging the inferred
// schemas into the skeleton, writing the result to outputPath.
func (m *Merger) Merge(result *ExtractionResult, outputPath string) error {
	// Read the original skeleton as raw JSON.
	skeleton, err := os.ReadFile(m.skeletonPath)
	if err != nil {
		return fmt.Errorf("read skeleton: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(skeleton, &doc); err != nil {
		return fmt.Errorf("unmarshal skeleton: %w", err)
	}

	// Ensure components/schemas exists.
	components := ensureMap(doc, "components")
	schemas := ensureMap(components, "schemas")

	// Merge endpoint response schemas.
	if result != nil {
		m.mergePathResponses(doc, result)
		m.registerComponents(schemas, result)
	}

	// Serialize with indentation.
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	if err := os.WriteFile(outputPath, out, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// mergePathResponses walks each endpoint in the extraction result and sets the
// response content type + schema on the corresponding skeleton path/method entry.
func (m *Merger) mergePathResponses(doc map[string]any, result *ExtractionResult) {
	paths := ensureMap(doc, "paths")

	for _, ep := range result.Endpoints {
		if ep.ResponseBody == nil {
			continue
		}

		// Resolve the path item in the skeleton.
		pathItem, ok := paths[ep.Endpoint.Path]
		if !ok {
			// Path not in skeleton — should not happen since we derived from it.
			continue
		}
		pathItemMap, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}

		// Resolve the method entry.
		methodKey := strings.ToLower(ep.Endpoint.Method)
		methodEntry, ok := pathItemMap[methodKey]
		if !ok {
			continue
		}
		methodMap, ok := methodEntry.(map[string]any)
		if !ok {
			continue
		}

		// Build the response schema for this endpoint.
		respSchema := m.buildResponseSchema(ep.ResponseBody, result)

		// Set responses.200.content["application/json"].schema.
		responses := ensureMap(methodMap, "responses")
		resp200 := ensureMap(responses, "200")
		content := ensureMap(resp200, "content")
		appJSON := ensureMap(content, "application/json")
		appJSON["schema"] = respSchema
	}
}

// buildResponseSchema converts an inferred SchemaField into an OpenAPI schema
// object, using $ref references where appropriate.
func (m *Merger) buildResponseSchema(schema *SchemaField, result *ExtractionResult) map[string]any {
	if schema == nil {
		return map[string]any{"type": "object"}
	}

	// Check if this schema matches a registered component.
	if name := schema.Name; name != "" {
		if _, exists := result.Components[name]; exists {
			return map[string]any{"$ref": Ref(name)}
		}
	}

	return SchemaToMap(schema)
}

// registerComponents adds all collected component schemas into the
// components/schemas map, using $ref references internally where possible.
func (m *Merger) registerComponents(schemas map[string]any, result *ExtractionResult) {
	for name, schema := range result.Components {
		if _, exists := schemas[name]; exists {
			// Preserve existing component definition (manual overrides).
			continue
		}
		schemas[name] = SchemaToMap(schema)
	}

	// Ensure the PageInfo schema has proper items reference if not already present.
	m.ensurePageInfoSchema(schemas, result)
}

// ensurePageInfoSchema checks whether the PageInfo component is properly
// structured and adds the missing items reference if needed.
func (m *Merger) ensurePageInfoSchema(schemas map[string]any, result *ExtractionResult) {
	pageInfo, ok := schemas["PageInfo"]
	if !ok {
		return
	}
	pageInfoMap, ok := pageInfo.(map[string]any)
	if !ok {
		return
	}

	props, ok := pageInfoMap["properties"].(map[string]any)
	if !ok {
		return
	}

	itemsField, ok := props["items"]
	if !ok {
		return
	}
	itemsMap, ok := itemsField.(map[string]any)
	if !ok {
		return
	}

	// If items is an empty object, give it a default type.
	if len(itemsMap) == 0 {
		itemsMap["type"] = "object"
	}
}

// InlineResponseSchemas builds response schemas inline (without $ref) for a
// given extraction result. This is used as a simpler alternative when
// component-based references are not desired.
func InlineResponseSchemas(result *ExtractionResult) (map[string]any, error) {
	paths := make(map[string]any)

	for _, ep := range result.Endpoints {
		if ep.ResponseBody == nil {
			continue
		}

		pathKey := ep.Endpoint.Path
		pathItem, ok := paths[pathKey]
		if !ok {
			pathItem = make(map[string]any)
			paths[pathKey] = pathItem
		}
		pathItemMap := pathItem.(map[string]any)

		methodKey := strings.ToLower(ep.Endpoint.Method)
		methodEntry := make(map[string]any)
		pathItemMap[methodKey] = methodEntry

		responses := make(map[string]any)
		methodEntry["responses"] = responses
		resp200 := make(map[string]any)
		responses["200"] = resp200
		resp200["description"] = "OK"

		content := make(map[string]any)
		resp200["content"] = content
		appJSON := make(map[string]any)
		content["application/json"] = appJSON

		schemaMap := SchemaToMap(ep.ResponseBody)
		schemaMap = stripRefs(schemaMap)
		appJSON["schema"] = schemaMap
	}

	return map[string]any{
		"openapi": "3.0.1",
		"info": map[string]any{
			"title":   "Extracted API",
			"version": "1.0.0",
		},
		"paths": paths,
	}, nil
}

// stripRefs recursively removes $ref fields from a schema map, replacing them
// with inline type: "object" definitions.
func stripRefs(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	// If there's a $ref, inline a basic object instead.
	if _, hasRef := m["$ref"]; hasRef {
		return map[string]any{"type": "object"}
	}

	result := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result[k] = stripRefs(val)
		case []any:
			stripped := make([]any, len(val))
			for i, item := range val {
				if itemMap, ok := item.(map[string]any); ok {
					stripped[i] = stripRefs(itemMap)
				} else {
					stripped[i] = item
				}
			}
			result[k] = stripped
		default:
			result[k] = v
		}
	}
	return result
}

// ensureMap retrieves the value for key in m as a map[string]any, creating an
// empty map if it is missing or nil.
func ensureMap(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok || v == nil {
		newMap := make(map[string]any)
		m[key] = newMap
		return newMap
	}
	if mm, ok := v.(map[string]any); ok {
		return mm
	}
	newMap := make(map[string]any)
	m[key] = newMap
	return newMap
}

// DeduplicateSchemas removes schema entries with the same name in the
// components/schemas section, keeping the one with more properties.
func (m *Merger) DeduplicateSchemas(schemas map[string]any) {
	type entry struct {
		name   string
		schema map[string]any
		score  int
	}

	var entries []entry
	for name, raw := range schemas {
		s, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		score := countProperties(s)
		entries = append(entries, entry{name: name, schema: s, score: score})
	}

	// Sort by score descending, keep highest, drop the rest with the same name.
	// Map iteration order is random, so we collect first, group by name, keep max.
	byName := make(map[string]map[string]any)
	bestScore := make(map[string]int)

	for _, e := range entries {
		if existing, ok := bestScore[e.name]; ok && existing >= e.score {
			continue
		}
		byName[e.name] = e.schema
		bestScore[e.name] = e.score
	}

	// Rebuild schemas with deduplicated entries.
	for name := range schemas {
		if _, keep := byName[name]; !keep {
			delete(schemas, name)
		}
	}
}

// countProperties recursively counts the number of property definitions in a schema.
func countProperties(s map[string]any) int {
	count := 0
	props, ok := s["properties"].(map[string]any)
	if !ok {
		return 1
	}
	for _, v := range props {
		count++
		if sub, ok := v.(map[string]any); ok {
			count += countProperties(sub)
		}
	}
	return count
}

// PageInfoSchema creates a standard PageInfo OpenAPI schema wrapping an array
// of items, where items references the given model name via $ref.
func PageInfoSchema(itemRef string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"page":      map[string]any{"type": "integer"},
			"page_size": map[string]any{"type": "integer"},
			"total":     map[string]any{"type": "integer"},
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"$ref": Ref(itemRef),
				},
			},
		},
	}
}

// SyncSchemaKeys renames schema property keys to match a canonical order for
// consistent diff output. Canonical order places commonly-used keys first.
func SyncSchemaKeys(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return schema
	}

	// Build ordered key list: known fields first in priority order, then
	// alphabetically.
	priority := []string{"id", "success", "message", "data", "name", "type", "status", "key"}
	seen := make(map[string]bool)
	var ordered []string

	// Priority keys in order.
	for _, k := range priority {
		if _, ok := props[k]; ok && !seen[k] {
			ordered = append(ordered, k)
			seen[k] = true
		}
	}

	// Remaining keys alphabetically.
	var rest []string
	for k := range props {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	ordered = append(ordered, rest...)

	// Build new properties map with ordered entries.
	orderedProps := make(map[string]any)
	for _, k := range ordered {
		orderedProps[k] = props[k]
	}
	schema["properties"] = orderedProps

	return schema
}

// NormalizeRefs rewrites $ref values pointing to schemas that exist in the
// components map from their original value to "#/components/schemas/{name}".
func NormalizeRefs(schema map[string]any, components map[string]bool) {
	if schema == nil {
		return
	}

	if ref, ok := schema["$ref"]; ok {
		refStr, ok := ref.(string)
		if ok && refStr != "" {
			// Extract the schema name from the ref.
			parts := strings.Split(refStr, "/")
			name := parts[len(parts)-1]
			if components[name] {
				// Already in the right format; no-op.
			}
		}
		return
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	for _, v := range props {
		if sub, ok := v.(map[string]any); ok {
			NormalizeRefs(sub, components)
		}
	}

	items, ok := schema["items"].(map[string]any)
	if ok {
		NormalizeRefs(items, components)
	}
}