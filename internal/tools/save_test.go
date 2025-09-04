package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/persistence"
	"github.com/dslh/mcp-metatool/internal/types"
)

func TestHandleSaveTool(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	tests := []struct {
		name        string
		args        types.SaveToolArgs
		wantSuccess bool
		wantError   string // substring that should appear in error message
	}{
		{
			"valid simple tool",
			types.SaveToolArgs{
				Name:        "test_tool",
				Description: "A simple test tool",
				InputSchema: map[string]interface{}{"type": "object"},
				Code:        "result = 'hello world'",
			},
			true,
			"",
		},
		{
			"valid complex tool",
			types.SaveToolArgs{
				Name:        "complex_tool",
				Description: "A complex tool with detailed schema",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]interface{}{
							"type":        "string",
							"description": "The message to process",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of repetitions",
							"minimum":     1,
						},
					},
					"required": []string{"message"},
				},
				Code: `message = params.get("message", "Hello")
count = params.get("count", 1)
result = [message] * count`,
			},
			true,
			"",
		},
		{
			"empty name",
			types.SaveToolArgs{
				Name:        "",
				Description: "Tool with empty name",
				Code:        "result = 'test'",
			},
			false,
			"tool name is required",
		},
		{
			"empty description",
			types.SaveToolArgs{
				Name:        "no_description_tool",
				Description: "",
				Code:        "result = 'test'",
			},
			false,
			"tool description is required",
		},
		{
			"empty code",
			types.SaveToolArgs{
				Name:        "no_code_tool",
				Description: "Tool with no code",
				Code:        "",
			},
			false,
			"tool code is required",
		},
		{
			"invalid tool name",
			types.SaveToolArgs{
				Name:        "invalid/name",
				Description: "Tool with invalid name",
				Code:        "result = 'test'",
			},
			false,
			"invalid character",
		},
		{
			"tool name too long",
			types.SaveToolArgs{
				Name:        strings.Repeat("a", 101),
				Description: "Tool with overly long name",
				Code:        "result = 'test'",
			},
			false,
			"too long",
		},
		{
			"tool with minimal schema",
			types.SaveToolArgs{
				Name:        "minimal_tool",
				Description: "Tool with minimal input schema",
				InputSchema: map[string]interface{}{},
				Code:        "result = 42",
			},
			true,
			"",
		},
		{
			"tool with nil schema",
			types.SaveToolArgs{
				Name:        "nil_schema_tool",
				Description: "Tool with nil input schema",
				InputSchema: nil,
				Code:        "result = 'works'",
			},
			true,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, returnValue, err := handleSaveTool(ctx, req, tt.args)

			// Check for framework errors
			if err != nil {
				t.Errorf("handleSaveTool() framework error = %v", err)
				return
			}

			// Check result structure
			if result == nil {
				t.Errorf("handleSaveTool() result is nil")
				return
			}

			if len(result.Content) == 0 {
				t.Errorf("handleSaveTool() result has no content")
				return
			}

			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Errorf("handleSaveTool() result content is not TextContent")
				return
			}

			if tt.wantSuccess {
				// Should contain success message
				expectedMsg := "saved successfully"
				if !strings.Contains(textContent.Text, expectedMsg) {
					t.Errorf("handleSaveTool() expected success message containing '%s', got: %s", expectedMsg, textContent.Text)
					return
				}

				// Return value should be the saved tool
				if returnValue == nil {
					t.Errorf("handleSaveTool() expected non-nil return value for success")
					return
				}

				savedTool, ok := returnValue.(*persistence.SavedToolDefinition)
				if !ok {
					t.Errorf("handleSaveTool() return value type = %T, want *persistence.SavedToolDefinition", returnValue)
					return
				}

				// Verify saved tool matches input
				if savedTool.Name != tt.args.Name {
					t.Errorf("handleSaveTool() saved tool name = %s, want %s", savedTool.Name, tt.args.Name)
				}
				if savedTool.Description != tt.args.Description {
					t.Errorf("handleSaveTool() saved tool description = %s, want %s", savedTool.Description, tt.args.Description)
				}
				if savedTool.Code != tt.args.Code {
					t.Errorf("handleSaveTool() saved tool code = %s, want %s", savedTool.Code, tt.args.Code)
				}

				// Verify tool was actually saved to disk
				toolsDir, _ := persistence.GetToolsDirectory()
				filename := filepath.Join(toolsDir, tt.args.Name+".json")
				if _, err := os.Stat(filename); os.IsNotExist(err) {
					t.Errorf("handleSaveTool() did not save tool to disk: %s", filename)
				}

				// Verify tool can be loaded back
				loadedTool, err := persistence.LoadTool(tt.args.Name)
				if err != nil {
					t.Errorf("handleSaveTool() saved tool cannot be loaded: %v", err)
				} else if loadedTool.Name != tt.args.Name {
					t.Errorf("handleSaveTool() loaded tool name = %s, want %s", loadedTool.Name, tt.args.Name)
				}
			} else {
				// Should contain error message
				if tt.wantError != "" && !strings.Contains(textContent.Text, tt.wantError) {
					t.Errorf("handleSaveTool() expected error containing '%s', got: %s", tt.wantError, textContent.Text)
				}

				// Return value should be nil for errors
				if returnValue != nil {
					t.Errorf("handleSaveTool() expected nil return value for error, got: %v", returnValue)
				}

				// Tool should not be saved to disk
				if tt.args.Name != "" && isValidToolName(tt.args.Name) {
					toolsDir, _ := persistence.GetToolsDirectory()
					filename := filepath.Join(toolsDir, tt.args.Name+".json")
					if _, err := os.Stat(filename); !os.IsNotExist(err) {
						t.Errorf("handleSaveTool() should not save invalid tool to disk")
					}
				}
			}
		})
	}
}

func TestHandleSaveToolOverwrite(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	// Save initial tool
	initialArgs := types.SaveToolArgs{
		Name:        "overwrite_test",
		Description: "Initial version",
		Code:        "result = 'version 1'",
	}

	result1, _, err1 := handleSaveTool(ctx, req, initialArgs)
	if err1 != nil {
		t.Fatalf("Initial save failed: %v", err1)
	}

	textContent1 := result1.Content[0].(*mcp.TextContent)
	if !strings.Contains(textContent1.Text, "saved successfully") {
		t.Fatalf("Initial save should succeed, got: %s", textContent1.Text)
	}

	// Overwrite with updated tool
	updatedArgs := types.SaveToolArgs{
		Name:        "overwrite_test",
		Description: "Updated version",
		Code:        "result = 'version 2'",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	}

	result2, returnValue2, err2 := handleSaveTool(ctx, req, updatedArgs)
	if err2 != nil {
		t.Fatalf("Overwrite save failed: %v", err2)
	}

	textContent2 := result2.Content[0].(*mcp.TextContent)
	if !strings.Contains(textContent2.Text, "saved successfully") {
		t.Fatalf("Overwrite save should succeed, got: %s", textContent2.Text)
	}

	// Verify the tool was updated
	savedTool := returnValue2.(*persistence.SavedToolDefinition)
	if savedTool.Description != updatedArgs.Description {
		t.Errorf("Overwritten tool description = %s, want %s", savedTool.Description, updatedArgs.Description)
	}
	if savedTool.Code != updatedArgs.Code {
		t.Errorf("Overwritten tool code = %s, want %s", savedTool.Code, updatedArgs.Code)
	}

	// Verify on disk
	loadedTool, err := persistence.LoadTool("overwrite_test")
	if err != nil {
		t.Fatalf("Failed to load overwritten tool: %v", err)
	}
	if loadedTool.Description != updatedArgs.Description {
		t.Errorf("Loaded overwritten tool description = %s, want %s", loadedTool.Description, updatedArgs.Description)
	}
}

func TestRegisterSaveTool(t *testing.T) {
	// Create a mock server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "0.1.0",
	}, nil)

	// Register the tool
	RegisterSaveTool(server)

	// Verify the registration doesn't panic
	t.Log("RegisterSaveTool completed without panic")
}

func TestSaveToolArgsValidation(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	tests := []struct {
		name      string
		args      types.SaveToolArgs
		wantError bool
	}{
		{
			"all fields valid",
			types.SaveToolArgs{
				Name:        "valid_tool",
				Description: "Valid tool description",
				Code:        "result = 'valid'",
				InputSchema: map[string]interface{}{"type": "object"},
			},
			false,
		},
		{
			"missing name",
			types.SaveToolArgs{
				Description: "Tool without name",
				Code:        "result = 'test'",
			},
			true,
		},
		{
			"missing description",
			types.SaveToolArgs{
				Name: "tool_no_desc",
				Code: "result = 'test'",
			},
			true,
		},
		{
			"missing code",
			types.SaveToolArgs{
				Name:        "tool_no_code",
				Description: "Tool without code",
			},
			true,
		},
		{
			"whitespace only name",
			types.SaveToolArgs{
				Name:        "   ",
				Description: "Tool with whitespace name",
				Code:        "result = 'test'",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, _, err := handleSaveTool(ctx, req, tt.args)

			if err != nil {
				t.Errorf("handleSaveTool() framework error = %v", err)
				return
			}

			textContent := result.Content[0].(*mcp.TextContent)
			containsError := strings.Contains(textContent.Text, "Error:") || strings.Contains(textContent.Text, "Failed")

			if tt.wantError && !containsError {
				t.Errorf("handleSaveTool() expected error, got: %s", textContent.Text)
			}
			if !tt.wantError && containsError {
				t.Errorf("handleSaveTool() unexpected error: %s", textContent.Text)
			}
		})
	}
}

func TestSaveToolIntegration(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	// Test complete workflow: save -> verify -> list -> load
	toolArgs := types.SaveToolArgs{
		Name:        "integration_test_tool",
		Description: "A tool for integration testing",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Code: `input_val = params.get("input", "default")
result = f"Processed: {input_val}"`,
	}

	// 1. Save the tool
	result, returnValue, err := handleSaveTool(ctx, req, toolArgs)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	textContent := result.Content[0].(*mcp.TextContent)
	if !strings.Contains(textContent.Text, "saved successfully") {
		t.Fatalf("Save should succeed, got: %s", textContent.Text)
	}

	savedTool := returnValue.(*persistence.SavedToolDefinition)

	// 2. Verify the tool structure
	if savedTool.Name != toolArgs.Name {
		t.Errorf("Saved tool name = %s, want %s", savedTool.Name, toolArgs.Name)
	}
	if savedTool.Description != toolArgs.Description {
		t.Errorf("Saved tool description = %s, want %s", savedTool.Description, toolArgs.Description)
	}
	if savedTool.Code != toolArgs.Code {
		t.Errorf("Saved tool code = %s, want %s", savedTool.Code, toolArgs.Code)
	}

	// 3. Verify tool appears in listing
	tools, err := persistence.ListTools()
	if err != nil {
		t.Fatalf("List tools failed: %v", err)
	}

	found := false
	for _, tool := range tools {
		if tool.Name == toolArgs.Name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Saved tool not found in listing")
	}

	// 4. Verify tool can be loaded independently
	loadedTool, err := persistence.LoadTool(toolArgs.Name)
	if err != nil {
		t.Fatalf("Load tool failed: %v", err)
	}

	if loadedTool.Name != toolArgs.Name {
		t.Errorf("Loaded tool name = %s, want %s", loadedTool.Name, toolArgs.Name)
	}
	if loadedTool.Description != toolArgs.Description {
		t.Errorf("Loaded tool description = %s, want %s", loadedTool.Description, toolArgs.Description)
	}
	if loadedTool.Code != toolArgs.Code {
		t.Errorf("Loaded tool code = %s, want %s", loadedTool.Code, toolArgs.Code)
	}

	// 5. Verify input schema is preserved
	if len(loadedTool.InputSchema) == 0 && len(toolArgs.InputSchema) > 0 {
		t.Errorf("Input schema was not preserved")
	}
}

// Helper functions

func isValidToolName(name string) bool {
	if name == "" || len(name) > 100 {
		return false
	}
	
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "..", " "}
	for _, char := range unsafe {
		if strings.Contains(name, char) {
			return false
		}
	}
	
	return true
}