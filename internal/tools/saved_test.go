package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/persistence"
	"github.com/dslh/mcp-metatool/internal/types"
)

// Test helper to create a saved tool with schema
func createTestToolWithSchema(t *testing.T, name, description, code string, schema map[string]interface{}) {
	t.Helper()
	
	tool := &persistence.SavedToolDefinition{
		Name:        name,
		Description: description,
		InputSchema: schema,
		Code:        code,
	}
	
	err := persistence.SaveTool(tool)
	if err != nil {
		t.Fatalf("Failed to create test tool %s: %v", name, err)
	}
}

func TestHandleSavedTool_ValidationIntegration(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	tests := []struct {
		name         string
		toolName     string
		toolCode     string
		schema       map[string]interface{}
		params       types.SavedToolParams
		expectError  bool
		errorContains string
		description  string
	}{
		{
			name:     "valid parameters pass validation",
			toolName: "greeting_tool",
			toolCode: `name = params.get("name", "World")
result = "Hello, " + name + "!"`,
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name to greet",
					},
				},
				"required": []interface{}{"name"},
			},
			params:      types.SavedToolParams{"name": "Alice"},
			expectError: false,
			description: "Valid parameters should pass validation and execute successfully",
		},
		{
			name:     "missing required parameter fails validation",
			toolName: "greeting_tool_required",
			toolCode: `name = params.get("name", "World")
result = "Hello, " + name + "!"`,
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name to greet",
					},
				},
				"required": []interface{}{"name"},
			},
			params:        types.SavedToolParams{},
			expectError:   true,
			errorContains: "Parameter validation failed",
			description:   "Missing required parameter should fail validation",
		},
		{
			name:     "wrong parameter type fails validation",
			toolName: "math_tool",
			toolCode: `x = params.get("number", 0)
result = x * 2`,
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"number": map[string]interface{}{
						"type":        "number",
						"description": "Number to double",
					},
				},
				"required": []interface{}{"number"},
			},
			params:        types.SavedToolParams{"number": "not a number"},
			expectError:   true,
			errorContains: "Parameter validation failed",
			description:   "Wrong parameter type should fail validation",
		},
		{
			name:     "empty schema allows any parameters",
			toolName: "flexible_tool",
			toolCode: `result = "Executed with params: " + str(params)`,
			schema:   map[string]interface{}{},
			params:   types.SavedToolParams{"anything": "goes", "number": 42},
			expectError: false,
			description: "Empty schema should allow any parameters",
		},
		{
			name:     "nil schema allows any parameters",
			toolName: "nil_schema_tool",
			toolCode: `result = "No schema validation"`,
			schema:   nil,
			params:   types.SavedToolParams{"whatever": "works"},
			expectError: false,
			description: "Nil schema should allow any parameters",
		},
		{
			name:     "complex nested object validation",
			toolName: "user_tool",
			toolCode: `user = params.get("user", {})
name = user.get("name", "Anonymous")
age = user.get("age", 0)
result = name + " is " + str(age) + " years old"`,
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
			params: types.SavedToolParams{
				"user": map[string]interface{}{
					"name": "Bob",
					"age":  30,
				},
			},
			expectError: false,
			description: "Complex nested object with valid data should pass",
		},
		{
			name:     "complex nested object validation failure",
			toolName: "user_tool_invalid",
			toolCode: `user = params.get("user", {})
result = "This should not execute"`,
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
			params: types.SavedToolParams{
				"user": map[string]interface{}{
					"name": "Bob",
					"age":  -5, // Invalid: negative age
				},
			},
			expectError:   true,
			errorContains: "Parameter validation failed",
			description:   "Complex nested object with invalid data should fail validation",
		},
		{
			name:     "array validation success",
			toolName: "list_processor",
			toolCode: `items = params.get("items", [])
result = "Processed " + str(len(items)) + " items: " + str(items)`,
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
			params:      types.SavedToolParams{"items": []interface{}{"item1", "item2", "item3"}},
			expectError: false,
			description: "Valid array parameter should pass validation",
		},
		{
			name:     "array validation failure",
			toolName: "list_processor_invalid",
			toolCode: `result = "Should not execute"`,
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
			params:        types.SavedToolParams{"items": []interface{}{"item1", 123}}, // 123 is not string
			expectError:   true,
			errorContains: "Parameter validation failed",
			description:   "Array with wrong item type should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the test tool
			createTestToolWithSchema(t, tt.toolName, "Test tool", tt.toolCode, tt.schema)
			
			// Load the tool
			tool, err := persistence.LoadTool(tt.toolName)
			if err != nil {
				t.Fatalf("Failed to load test tool: %v", err)
			}

			// Execute the tool
			result, _, err := handleSavedTool(tool, tt.params)
			
			if tt.expectError {
				if result == nil || len(result.Content) == 0 {
					t.Errorf("%s: expected error result, got nil", tt.description)
					return
				}
				
				// Check that the error message is in the content
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Errorf("%s: expected TextContent, got %T", tt.description, result.Content[0])
					return
				}
				
				if tt.errorContains != "" && !strings.Contains(textContent.Text, tt.errorContains) {
					t.Errorf("%s: expected error containing '%s', got '%s'", tt.description, tt.errorContains, textContent.Text)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
					return
				}
				
				if result == nil {
					t.Errorf("%s: expected successful result, got nil", tt.description)
					return
				}
				
				// Verify we have content (successful execution)
				if len(result.Content) == 0 {
					t.Errorf("%s: expected content in successful result", tt.description)
				}
			}
		})
	}
}

func TestHandleSavedTool_RuntimeErrorsAfterValidation(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	// Create a tool that passes validation but has a runtime error
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []interface{}{"name"},
	}
	
	// This Starlark code has a runtime error (undefined variable)
	createTestToolWithSchema(t, "runtime_error_tool", "Tool with runtime error", 
		`result = undefined_variable + " test"`, schema)
	
	tool, err := persistence.LoadTool("runtime_error_tool")
	if err != nil {
		t.Fatalf("Failed to load test tool: %v", err)
	}

	// Valid parameters should pass validation but then hit runtime error
	params := types.SavedToolParams{"name": "test"}
	result, _, err := handleSavedTool(tool, params)
	
	// Should not return Go error, but should have error in result content
	if err != nil {
		t.Errorf("Expected no Go error, got: %v", err)
	}
	
	if result == nil || len(result.Content) == 0 {
		t.Fatal("Expected result with error content")
	}
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}
	
	// Should contain runtime error message, not validation error
	if strings.Contains(textContent.Text, "Parameter validation failed") {
		t.Errorf("Should not contain validation error, got: %s", textContent.Text)
	}
	
	if !strings.Contains(textContent.Text, "Tool execution failed") && !strings.Contains(textContent.Text, "Tool error") {
		t.Errorf("Expected runtime error message, got: %s", textContent.Text)
	}
}