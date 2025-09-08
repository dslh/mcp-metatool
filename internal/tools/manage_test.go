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

// Test helper to create a test tool on disk
func createTestTool(t *testing.T, name, description, code string) {
	t.Helper()
	
	tool := &persistence.SavedToolDefinition{
		Name:        name,
		Description: description,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Test input",
				},
			},
		},
		Code: code,
	}
	
	err := persistence.SaveTool(tool)
	if err != nil {
		t.Fatalf("Failed to create test tool %s: %v", name, err)
	}
}

// Test helper to verify ToolListResponse structure
func verifyToolListResponse(t *testing.T, returnValue interface{}, expectedCount int) *ToolListResponse {
	t.Helper()
	
	if returnValue == nil {
		t.Fatalf("Expected ToolListResponse, got nil")
	}
	
	response, ok := returnValue.(ToolListResponse)
	if !ok {
		t.Fatalf("Expected ToolListResponse, got %T", returnValue)
	}
	
	if len(response.Tools) != expectedCount {
		t.Errorf("Expected %d tools, got %d", expectedCount, len(response.Tools))
	}
	
	return &response
}

// Test helper to verify text content contains expected substring
func verifyTextContent(t *testing.T, result *mcp.CallToolResult, expectedSubstring string) {
	t.Helper()
	
	if result == nil {
		t.Fatalf("CallToolResult is nil")
	}
	
	if len(result.Content) == 0 {
		t.Fatalf("CallToolResult has no content")
	}
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}
	
	if !strings.Contains(textContent.Text, expectedSubstring) {
		t.Errorf("Expected text containing '%s', got: %s", expectedSubstring, textContent.Text)
	}
}

func TestHandleListSavedTools(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	tests := []struct {
		name          string
		setupTools    []struct{ name, description, code string }
		expectedCount int
		expectedText  string // substring that should appear in response text
	}{
		{
			"no tools",
			[]struct{ name, description, code string }{},
			0,
			"No saved tools found",
		},
		{
			"single tool",
			[]struct{ name, description, code string }{
				{"test_tool", "A simple test tool", "result = 'hello'"},
			},
			1,
			"Found 1 saved tool(s)",
		},
		{
			"multiple tools",
			[]struct{ name, description, code string }{
				{"tool_one", "First test tool", "result = 1"},
				{"tool_two", "Second test tool", "result = 2"},
				{"tool_three", "Third test tool", "result = 3"},
			},
			3,
			"Found 3 saved tool(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing tools
			toolsDir, _ := persistence.GetToolsDirectory()
			os.RemoveAll(toolsDir)

			// Create test tools
			for _, tool := range tt.setupTools {
				createTestTool(t, tool.name, tool.description, tool.code)
			}

			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, returnValue, err := handleListSavedTools(ctx, req, struct{}{})

			// Check for framework errors
			if err != nil {
				t.Errorf("handleListSavedTools() framework error = %v", err)
				return
			}

			// Verify text content
			verifyTextContent(t, result, tt.expectedText)

			// Verify return value structure
			response := verifyToolListResponse(t, returnValue, tt.expectedCount)

			// For non-empty cases, verify tool details in response
			if tt.expectedCount > 0 {
				// Verify that each setup tool appears in the response
				responseToolNames := make(map[string]string)
				for _, tool := range response.Tools {
					responseToolNames[tool.Name] = tool.Description
				}

				for _, setupTool := range tt.setupTools {
					if desc, found := responseToolNames[setupTool.name]; !found {
						t.Errorf("Expected tool %s not found in response", setupTool.name)
					} else if desc != setupTool.description {
						t.Errorf("Tool %s description = %s, want %s", setupTool.name, desc, setupTool.description)
					}
				}

				// Verify readable format includes tool names and descriptions
				textContent := result.Content[0].(*mcp.TextContent)
				for _, setupTool := range tt.setupTools {
					expectedLine := "• " + setupTool.name + ": " + setupTool.description
					if !strings.Contains(textContent.Text, expectedLine) {
						t.Errorf("Expected readable format to contain '%s'", expectedLine)
					}
				}
			}
		})
	}
}

func TestHandleShowSavedTool(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	// Create a test tool
	testToolName := "show_test_tool"
	testToolDescription := "A tool for testing show functionality"
	testToolCode := "result = 'show test'"
	createTestTool(t, testToolName, testToolDescription, testToolCode)

	tests := []struct {
		name        string
		args        types.ShowToolArgs
		wantSuccess bool
		wantError   string // substring that should appear in error message
	}{
		{
			"valid tool name",
			types.ShowToolArgs{Name: testToolName},
			true,
			"",
		},
		{
			"empty tool name",
			types.ShowToolArgs{Name: ""},
			false,
			"tool name is required",
		},
		{
			"non-existent tool",
			types.ShowToolArgs{Name: "does_not_exist"},
			false,
			"Failed to load tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, returnValue, err := handleShowSavedTool(ctx, req, tt.args)

			// Check for framework errors
			if err != nil {
				t.Errorf("handleShowSavedTool() framework error = %v", err)
				return
			}

			// Check result structure
			if result == nil {
				t.Errorf("handleShowSavedTool() result is nil")
				return
			}

			if len(result.Content) == 0 {
				t.Errorf("handleShowSavedTool() result has no content")
				return
			}

			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Errorf("handleShowSavedTool() result content is not TextContent")
				return
			}

			if tt.wantSuccess {
				// Should return the tool's Starlark code
				if textContent.Text != testToolCode {
					t.Errorf("handleShowSavedTool() expected Starlark code '%s', got: %s", testToolCode, textContent.Text)
					return
				}

				// Return value should be the tool definition
				if returnValue == nil {
					t.Errorf("handleShowSavedTool() expected non-nil return value for success")
					return
				}

				tool, ok := returnValue.(*persistence.SavedToolDefinition)
				if !ok {
					t.Errorf("handleShowSavedTool() return value type = %T, want *persistence.SavedToolDefinition", returnValue)
					return
				}

				// Verify tool matches expected values
				if tool.Name != testToolName {
					t.Errorf("handleShowSavedTool() tool name = %s, want %s", tool.Name, testToolName)
				}
				if tool.Description != testToolDescription {
					t.Errorf("handleShowSavedTool() tool description = %s, want %s", tool.Description, testToolDescription)
				}
				if tool.Code != testToolCode {
					t.Errorf("handleShowSavedTool() tool code = %s, want %s", tool.Code, testToolCode)
				}
			} else {
				// Should contain error message
				if tt.wantError != "" && !strings.Contains(textContent.Text, tt.wantError) {
					t.Errorf("handleShowSavedTool() expected error containing '%s', got: %s", tt.wantError, textContent.Text)
				}

				// Return value should be nil for error cases
				if returnValue != nil {
					t.Errorf("handleShowSavedTool() expected nil return value for error case, got: %v", returnValue)
				}
			}
		})
	}
}

func TestHandleDeleteSavedTool(t *testing.T) {
	// Setup temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	tests := []struct {
		name        string
		args        types.DeleteToolArgs
		setupTool   bool // whether to create the test tool before deletion
		wantSuccess bool
		wantError   string // substring that should appear in error message
	}{
		{
			"valid tool deletion",
			types.DeleteToolArgs{Name: "delete_test_tool"},
			true,
			true,
			"",
		},
		{
			"empty tool name",
			types.DeleteToolArgs{Name: ""},
			false,
			false,
			"tool name is required",
		},
		{
			"non-existent tool",
			types.DeleteToolArgs{Name: "does_not_exist"},
			false,
			false,
			"does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing tools
			toolsDir, _ := persistence.GetToolsDirectory()
			os.RemoveAll(toolsDir)

			// Create test tool if needed
			if tt.setupTool {
				createTestTool(t, tt.args.Name, "A tool for testing delete functionality", "result = 'delete test'")
			}

			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, returnValue, err := handleDeleteSavedTool(ctx, req, tt.args)

			// Check for framework errors
			if err != nil {
				t.Errorf("handleDeleteSavedTool() framework error = %v", err)
				return
			}

			// Check result structure
			if result == nil {
				t.Errorf("handleDeleteSavedTool() result is nil")
				return
			}

			if len(result.Content) == 0 {
				t.Errorf("handleDeleteSavedTool() result has no content")
				return
			}

			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Errorf("handleDeleteSavedTool() result content is not TextContent")
				return
			}

			if tt.wantSuccess {
				// Should contain success message
				expectedMsg := "deleted successfully"
				if !strings.Contains(textContent.Text, expectedMsg) {
					t.Errorf("handleDeleteSavedTool() expected success message containing '%s', got: %s", expectedMsg, textContent.Text)
					return
				}

				// Should mention restart requirement
				if !strings.Contains(textContent.Text, "Restart server") {
					t.Errorf("handleDeleteSavedTool() expected restart message in: %s", textContent.Text)
				}

				// Return value should contain deletion confirmation
				if returnValue == nil {
					t.Errorf("handleDeleteSavedTool() expected non-nil return value for success")
					return
				}

				deletionInfo, ok := returnValue.(map[string]string)
				if !ok {
					t.Errorf("handleDeleteSavedTool() return value type = %T, want map[string]string", returnValue)
					return
				}

				if deletionInfo["deleted"] != tt.args.Name {
					t.Errorf("handleDeleteSavedTool() deletion info = %v, want deleted: %s", deletionInfo, tt.args.Name)
				}

				// Verify tool was actually deleted from disk
				toolsDir, _ := persistence.GetToolsDirectory()
				filename := filepath.Join(toolsDir, tt.args.Name+".json")
				if _, err := os.Stat(filename); !os.IsNotExist(err) {
					t.Errorf("handleDeleteSavedTool() tool file still exists: %s", filename)
				}

				// Verify tool cannot be loaded anymore
				_, err := persistence.LoadTool(tt.args.Name)
				if err == nil {
					t.Errorf("handleDeleteSavedTool() deleted tool can still be loaded")
				}
			} else {
				// Should contain error message
				if tt.wantError != "" && !strings.Contains(textContent.Text, tt.wantError) {
					t.Errorf("handleDeleteSavedTool() expected error containing '%s', got: %s", tt.wantError, textContent.Text)
				}

				// Return value should be nil for error cases
				if returnValue != nil {
					t.Errorf("handleDeleteSavedTool() expected nil return value for error case, got: %v", returnValue)
				}
			}
		})
	}
}