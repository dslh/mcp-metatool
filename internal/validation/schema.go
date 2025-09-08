package validation

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// ValidationError represents a parameter validation error
type ValidationError struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return e.Message
}

// ValidateParams validates parameters against a JSON Schema
func ValidateParams(schema map[string]interface{}, params map[string]interface{}) error {
	// Handle empty schema case - if no schema is provided, accept any parameters
	if len(schema) == 0 {
		return nil
	}

	// Marshal the schema map to JSON and unmarshal into Schema struct
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return &ValidationError{
			Type:    "SchemaError",
			Message: "Failed to marshal schema",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	var schemaObj jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
		return &ValidationError{
			Type:    "SchemaError",
			Message: "Invalid JSON schema definition",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Resolve the schema
	resolved, err := schemaObj.Resolve(nil)
	if err != nil {
		return &ValidationError{
			Type:    "SchemaError",
			Message: "Failed to resolve JSON schema",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Validate the parameters
	if err := resolved.Validate(params); err != nil {
		return &ValidationError{
			Type:    "ValidationError",
			Message: "Parameter validation failed",
			Details: map[string]interface{}{
				"error":          err.Error(),
				"schema":         schema,
				"providedParams": params,
			},
		}
	}

	return nil
}

// FormatValidationError formats a validation error for display
func FormatValidationError(err error) string {
	if validationErr, ok := err.(*ValidationError); ok {
		if len(validationErr.Details) > 0 {
			if errorMsg, hasError := validationErr.Details["error"].(string); hasError {
				return fmt.Sprintf("%s: %s", validationErr.Message, errorMsg)
			}
		}
		return validationErr.Message
	}
	return err.Error()
}