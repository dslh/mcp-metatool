package schema

import (
	"log"

	"github.com/google/jsonschema-go/jsonschema"
)

// Transform converts JSON Schema draft-07 to draft-2020-12 for compatibility
// This handles the main compatibility issue between different schema versions
func Transform(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return nil
	}

	// Create a copy to avoid modifying the original
	transformed := *schema

	// Handle the main compatibility issue: transform draft-07 $schema to draft-2020-12
	if schema.Schema == "http://json-schema.org/draft-07/schema#" ||
		schema.Schema == "http://json-schema.org/draft-07/schema" {
		transformed.Schema = "https://json-schema.org/draft/2020-12/schema"
	}

	// Recursively transform nested schemas in properties
	if schema.Properties != nil {
		transformed.Properties = make(map[string]*jsonschema.Schema)
		for k, v := range schema.Properties {
			transformed.Properties[k] = Transform(v)
		}
	}

	// Transform items schema if present
	if schema.Items != nil {
		transformed.Items = Transform(schema.Items)
	}

	// Transform additional properties schema if present
	if schema.AdditionalProperties != nil {
		transformed.AdditionalProperties = Transform(schema.AdditionalProperties)
	}

	return &transformed
}

// SafeTransform safely transforms a schema with error handling
// Returns nil if transformation fails, allowing graceful degradation
func SafeTransform(schema *jsonschema.Schema, context string) *jsonschema.Schema {
	var result *jsonschema.Schema
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Warning: Schema transformation failed for %s: %v. Proceeding without schema validation.", context, r)
			result = nil
		}
	}()

	result = Transform(schema)
	return result
}