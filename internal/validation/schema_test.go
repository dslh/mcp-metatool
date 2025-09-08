package validation

import (
	"strings"
	"testing"
)

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name        string
		schema      map[string]interface{}
		params      map[string]interface{}
		expectError bool
		errorType   string
		description string
	}{
		{
			name:        "empty schema allows any params",
			schema:      map[string]interface{}{},
			params:      map[string]interface{}{"anything": "goes"},
			expectError: false,
			description: "Empty schema should accept any parameters",
		},
		{
			name:   "valid simple string parameter",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"name"},
			},
			params:      map[string]interface{}{"name": "test"},
			expectError: false,
			description: "Valid string parameter should pass validation",
		},
		{
			name:   "missing required parameter",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"name"},
			},
			params:      map[string]interface{}{},
			expectError: true,
			errorType:   "ValidationError",
			description: "Missing required parameter should fail validation",
		},
		{
			name:   "wrong parameter type",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"age": map[string]interface{}{
						"type": "number",
					},
				},
			},
			params:      map[string]interface{}{"age": "not a number"},
			expectError: true,
			errorType:   "ValidationError",
			description: "Wrong parameter type should fail validation",
		},
		{
			name:   "complex nested schema validation",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type": "string",
							},
							"age": map[string]interface{}{
								"type":    "number",
								"minimum": 0,
							},
						},
						"required": []interface{}{"name", "age"},
					},
				},
				"required": []interface{}{"user"},
			},
			params: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  25,
				},
			},
			expectError: false,
			description: "Valid nested object should pass validation",
		},
		{
			name:   "complex nested schema with validation error",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type": "string",
							},
							"age": map[string]interface{}{
								"type":    "number",
								"minimum": 0,
							},
						},
						"required": []interface{}{"name", "age"},
					},
				},
				"required": []interface{}{"user"},
			},
			params: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  -5, // Invalid: negative age
				},
			},
			expectError: true,
			errorType:   "ValidationError",
			description: "Nested object with validation constraint violation should fail",
		},
		{
			name:   "array parameter validation",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"items": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"minItems": 1,
					},
				},
				"required": []interface{}{"items"},
			},
			params: map[string]interface{}{
				"items": []interface{}{"item1", "item2"},
			},
			expectError: false,
			description: "Valid array parameter should pass validation",
		},
		{
			name:   "array parameter with wrong item type",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"items": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			params: map[string]interface{}{
				"items": []interface{}{"item1", 123}, // 123 is not a string
			},
			expectError: true,
			errorType:   "ValidationError",
			description: "Array with wrong item type should fail validation",
		},
		{
			name: "invalid JSON schema definition",
			schema: map[string]interface{}{
				"type":       "object",
				"properties": "this should be an object, not a string",
			},
			params:      map[string]interface{}{"name": "test"},
			expectError: true,
			errorType:   "SchemaError",
			description: "Invalid schema definition should return SchemaError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParams(tt.schema, tt.params)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
					return
				}
				
				// Check error type if specified
				if tt.errorType != "" {
					if validationErr, ok := err.(*ValidationError); ok {
						if validationErr.Type != tt.errorType {
							t.Errorf("%s: expected error type %s, got %s", tt.description, tt.errorType, validationErr.Type)
						}
					} else {
						t.Errorf("%s: expected ValidationError, got %T", tt.description, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("%s: expected no error but got: %v", tt.description, err)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Type:    "ValidationError",
		Message: "Test error message",
		Details: map[string]interface{}{
			"field": "value",
		},
	}

	// Test Error() method
	if err.Error() != "Test error message" {
		t.Errorf("Expected Error() to return 'Test error message', got '%s'", err.Error())
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedSubstr string
		description    string
	}{
		{
			name: "validation error with details",
			err: &ValidationError{
				Type:    "ValidationError",
				Message: "Parameter validation failed",
				Details: map[string]interface{}{
					"error": "missing required property: name",
				},
			},
			expectedSubstr: "Parameter validation failed: missing required property: name",
			description:    "Should format validation error with error detail",
		},
		{
			name: "validation error without details",
			err: &ValidationError{
				Type:    "ValidationError",
				Message: "Basic validation error",
			},
			expectedSubstr: "Basic validation error",
			description:    "Should return basic message for error without details",
		},
		{
			name:           "generic error",
			err:            strings.NewReader("").UnreadRune(), // Creates a generic error
			expectedSubstr: "",
			description:    "Should handle generic errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValidationError(tt.err)
			
			if tt.expectedSubstr != "" && !strings.Contains(result, tt.expectedSubstr) {
				t.Errorf("%s: expected result to contain '%s', got '%s'", tt.description, tt.expectedSubstr, result)
			}
			
			// Ensure result is not empty for any error
			if result == "" {
				t.Errorf("%s: result should not be empty", tt.description)
			}
		})
	}
}