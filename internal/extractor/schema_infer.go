package extractor

import (
	"fmt"
	"strings"
	"time"
)

// InferSchema converts a generic JSON value (from json.Unmarshal into any)
// into an inferred SchemaField tree.
func InferSchema(data any) *SchemaField {
	if data == nil {
		return &SchemaField{Type: "string", Nullable: true}
	}

	switch v := data.(type) {
	case bool:
		return &SchemaField{Type: "boolean"}
	case float64:
		// Check if it looks like a whole number.
		if v == float64(int64(v)) {
			return &SchemaField{Type: "integer"}
		}
		return &SchemaField{Type: "number"}
	case string:
		sf := &SchemaField{Type: "string"}
		if format := detectStringFormat(v); format != "" {
			sf.Format = format
		}
		return sf
	case []any:
		return inferArray(v)
	case map[string]any:
		return inferObject(v)
	default:
		return &SchemaField{Type: "string"}
	}
}

// inferArray infers a schema for a JSON array by sampling the first element.
func inferArray(data []any) *SchemaField {
	sf := &SchemaField{Type: "array"}
	if len(data) > 0 {
		items := InferSchema(data[0])
		sf.Items = items

		// Collect unique enum values if all elements are the same primitive type.
		collectEnumValues(data, sf)
	}
	return sf
}

// inferObject infers a schema for a JSON object by analyzing each field.
func inferObject(data map[string]any) *SchemaField {
	sf := &SchemaField{Type: "object"}

	// Track which field names we've seen for enum detection and nullable marking.
	for key, val := range data {
		field := InferSchema(val)
		field.Name = key

		// Mark zero-values as nullable (empty string, 0, false).
		if isZeroValue(val) {
			field.Nullable = true
		}

		sf.Properties = append(sf.Properties, *field)
	}

	// Attempt to assign a meaningful component name based on property patterns.
	sf.Name = inferComponentName(sf)

	return sf
}

// isZeroValue returns true when val is the zero value for its type.
func isZeroValue(val any) bool {
	switch v := val.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case float64:
		return v == 0
	case bool:
		return !v
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}

// detectStringFormat tries to identify well-known string formats.
func detectStringFormat(s string) string {
	if s == "" {
		return ""
	}

	// RFC3339 / ISO 8601 datetime.
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return "date-time"
	}
	// ISO 8601 without timezone.
	if _, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return "date-time"
	}
	// Date only.
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return "date"
	}

	return ""
}

// collectEnumValues examines an array of homogeneous scalar values and, if the
// set is small enough, records an enum constraint on the parent array schema.
func collectEnumValues(data []any, parent *SchemaField) {
	if len(data) == 0 || data == nil {
		return
	}

	// Only collect enum from string and number arrays, and only when the
	// sample size is reasonable (<= 100 elements).
	if len(data) > 100 {
		return
	}

	seen := make(map[any]struct{})
	var unique []any
	for _, item := range data {
		switch item.(type) {
		case string, float64, bool:
			if _, ok := seen[item]; !ok {
				seen[item] = struct{}{}
				unique = append(unique, item)
			}
		default:
			return // non-scalar element — don't try enum.
		}
	}

	// Only emit enum if the variety is small relative to the sample.
	if len(unique) > 1 && len(unique) <= 10 {
		parent.Items.Enum = unique
	}
}

// inferComponentName attempts to derive a meaningful schema name from an
// object's property set by matching common field patterns.
func inferComponentName(schema *SchemaField) string {
	if schema == nil || schema.Type != "object" {
		return ""
	}

	props := make(map[string]bool)
	for _, p := range schema.Properties {
		props[strings.ToLower(p.Name)] = true
	}

	// Match common model signatures by field presence.
	switch {
	case props["id"] && props["name"] && props["type"] && props["status"] && props["models"]:
		return "Channel"
	case props["id"] && props["username"] && props["role"] && props["status"] && props["quota"]:
		return "User"
	case props["id"] && props["user_id"] && props["type"] && props["content"]:
		return "Log"
	case props["id"] && props["user_id"] && props["key"] && props["name"] && props["status"]:
		return "Token"
	case props["id"] && props["name"] && props["key"] && props["status"] && props["quota"]:
		return "Redemption"
	case props["success"] && props["message"] && !props["data"]:
		return "ApiResponse"
	case props["success"] && props["message"] && props["data"]:
		return "ApiResponseWithData"
	case props["page"] && props["page_size"] && props["total"] && props["items"]:
		return "PageInfo"
	default:
		return ""
	}
}

// MergeFields merges two schema fields, preferring non-nullable over nullable
// and preserving the richer type (object > string > ...).
func MergeFields(existing, inferred *SchemaField) *SchemaField {
	if existing == nil {
		return inferred
	}
	if inferred == nil {
		return existing
	}

	merged := *existing

	// If the existing nullable but inferred is not, use non-nullable.
	if existing.Nullable && !inferred.Nullable {
		merged.Nullable = false
	}
	// If the existing has no type but inferred has one, use inferred.
	if existing.Type == "" && inferred.Type != "" {
		merged.Type = inferred.Type
	}
	// Merge format.
	if existing.Format == "" && inferred.Format != "" {
		merged.Format = inferred.Format
	}
	// Preserve description if inferred has one.
	if existing.Description == "" && inferred.Description != "" {
		merged.Description = inferred.Description
	}
	// Merge properties for objects.
	if existing.Type == "object" && inferred.Type == "object" {
		mergedProps := mergePropertyLists(existing.Properties, inferred.Properties)
		merged.Properties = mergedProps
	}
	// Merge array items.
	if existing.Type == "array" && inferred.Type == "array" {
		merged.Items = MergeFields(existing.Items, inferred.Items)
	}

	return &merged
}

// mergePropertyLists merges two slices of properties by name.
func mergePropertyLists(existing, inferred []SchemaField) []SchemaField {
	byName := make(map[string]*SchemaField, len(existing))
	for i := range existing {
		byName[existing[i].Name] = &existing[i]
	}

	for i := range inferred {
		name := inferred[i].Name
		if existingField, ok := byName[name]; ok {
			merged := MergeFields(existingField, &inferred[i])
			byName[name] = merged
		} else {
			// New field from inferred data.
			cp := inferred[i]
			byName[name] = &cp
		}
	}

	result := make([]SchemaField, 0, len(byName))
	for _, sf := range byName {
		result = append(result, *sf)
	}
	return result
}

// SchemaToMap converts a SchemaField tree into a map suitable for JSON
// serialization as an OpenAPI schema object.
func SchemaToMap(schema *SchemaField) map[string]any {
	if schema == nil {
		return map[string]any{}
	}

	m := map[string]any{}

	if schema.Ref != "" {
		m["$ref"] = schema.Ref
		return m
	}

	if schema.Type != "" {
		m["type"] = schema.Type
	}
	if schema.Format != "" {
		m["format"] = schema.Format
	}
	if schema.Nullable {
		m["nullable"] = true
	}
	if schema.Description != "" {
		m["description"] = schema.Description
	}
	if len(schema.Enum) > 0 {
		m["enum"] = schema.Enum
	}

	if schema.Type == "object" && len(schema.Properties) > 0 {
		props := make(map[string]any)
		for _, prop := range schema.Properties {
			props[prop.Name] = SchemaToMap(&prop)
		}
		m["properties"] = props
	}

	if schema.Type == "array" && schema.Items != nil {
		m["items"] = SchemaToMap(schema.Items)
	}

	return m
}

// ValidateOpenSchema ensures a schema map has at least a type or properties.
func ValidateOpenSchema(m map[string]any) bool {
	if m == nil {
		return false
	}
	if _, ok := m["type"]; ok {
		return true
	}
	if _, ok := m["properties"]; ok {
		return true
	}
	if _, ok := m["$ref"]; ok {
		return true
	}
	if _, ok := m["items"]; ok {
		return true
	}
	return false
}

// FormatName converts a path segment into a PascalCase OpenAPI schema name.
// Example: "/api/channel/" -> "Channel", "log" -> "Log".
func FormatName(segment string) string {
	if segment == "" {
		return ""
	}
	// Strip { } from path params.
	segment = strings.NewReplacer("{", "", "}", "").Replace(segment)
	// PascalCase.
	parts := strings.FieldsFunc(segment, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	result := b.String()
	if result == "" {
		return ""
	}
	return result
}

// Ref creates a $ref string from a schema name.
func Ref(name string) string {
	return fmt.Sprintf("#/components/schemas/%s", name)
}

// SortPropertiesByName sorts schema properties in place by their Name field.
func SortPropertiesByName(props []SchemaField) {
	// Simple insertion sort — Go 1.25 doesn't guarantee slices.SortFunc in all
	// build environments, and the property lists are small (< 50 fields).
	for i := 1; i < len(props); i++ {
		for j := i; j > 0 && props[j-1].Name > props[j].Name; j-- {
			props[j], props[j-1] = props[j-1], props[j]
		}
	}
}