package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"valid simple", "my_tool", false},
		{"valid with numbers", "tool123", false},
		{"valid with underscore", "my_cool_tool", false},
		{"valid with hyphen", "my-tool", false},
		{"valid mixed", "tool_123-abc", false},
		{"single char", "a", false},
		{"max length", strings.Repeat("a", 100), false},
		
		// Invalid names
		{"empty", "", true},
		{"too long", strings.Repeat("a", 101), true},
		{"with slash", "tool/name", true},
		{"with backslash", "tool\\name", true},
		{"with colon", "tool:name", true},
		{"with asterisk", "tool*name", true},
		{"with question", "tool?name", true},
		{"with quote", "tool\"name", true},
		{"with less than", "tool<name", true},
		{"with greater than", "tool>name", true},
		{"with pipe", "tool|name", true},
		{"with double dot", "tool..name", true},
		{"with space", "tool name", true},
		{"only spaces", "   ", true},
		{"unicode", "tool🚀", false}, // Should be allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestGetToolsDirectory(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("MCP_METATOOL_DIR")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("MCP_METATOOL_DIR")
		} else {
			os.Setenv("MCP_METATOOL_DIR", originalEnv)
		}
	}()

	t.Run("default path", func(t *testing.T) {
		os.Unsetenv("MCP_METATOOL_DIR")
		
		dir, err := GetToolsDirectory()
		if err != nil {
			t.Errorf("GetToolsDirectory() error = %v", err)
			return
		}
		
		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ".mcp-metatool", "tools")
		if dir != expected {
			t.Errorf("GetToolsDirectory() = %q, want %q", dir, expected)
		}
		
		// Check that directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("GetToolsDirectory() did not create directory: %s", dir)
		}
	})

	t.Run("environment override", func(t *testing.T) {
		tempDir := t.TempDir()
		customDir := filepath.Join(tempDir, "custom-metatool")
		os.Setenv("MCP_METATOOL_DIR", customDir)
		
		dir, err := GetToolsDirectory()
		if err != nil {
			t.Errorf("GetToolsDirectory() error = %v", err)
			return
		}
		
		expected := filepath.Join(customDir, "tools")
		if dir != expected {
			t.Errorf("GetToolsDirectory() = %q, want %q", dir, expected)
		}
		
		// Check that directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("GetToolsDirectory() did not create directory: %s", dir)
		}
	})
}

func TestSaveTool(t *testing.T) {
	// Setup temp directory
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	tests := []struct {
		name    string
		tool    *SavedToolDefinition
		wantErr bool
	}{
		{
			"valid tool",
			&SavedToolDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: map[string]interface{}{"type": "object"},
				Code:        "result = 'hello'",
			},
			false,
		},
		{
			"empty name",
			&SavedToolDefinition{
				Name:        "",
				Description: "A test tool",
				Code:        "result = 'hello'",
			},
			true,
		},
		{
			"invalid name",
			&SavedToolDefinition{
				Name:        "invalid/name",
				Description: "A test tool",
				Code:        "result = 'hello'",
			},
			true,
		},
		{
			"complex tool",
			&SavedToolDefinition{
				Name:        "complex_tool",
				Description: "A complex tool with schema",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Code: `name = params.get("name", "World")
result = f"Hello, {name}!"`,
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveTool(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Verify file was created
				toolsDir, _ := GetToolsDirectory()
				filename := filepath.Join(toolsDir, tt.tool.Name+".json")
				if _, err := os.Stat(filename); os.IsNotExist(err) {
					t.Errorf("SaveTool() did not create file: %s", filename)
					return
				}
				
				// Verify file contents
				data, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("SaveTool() could not read saved file: %v", err)
					return
				}
				
				var savedTool SavedToolDefinition
				if err := json.Unmarshal(data, &savedTool); err != nil {
					t.Errorf("SaveTool() saved invalid JSON: %v", err)
					return
				}
				
				if savedTool.Name != tt.tool.Name {
					t.Errorf("SaveTool() saved name = %q, want %q", savedTool.Name, tt.tool.Name)
				}
				if savedTool.Description != tt.tool.Description {
					t.Errorf("SaveTool() saved description = %q, want %q", savedTool.Description, tt.tool.Description)
				}
				if savedTool.Code != tt.tool.Code {
					t.Errorf("SaveTool() saved code = %q, want %q", savedTool.Code, tt.tool.Code)
				}
			}
		})
	}
}

func TestLoadTool(t *testing.T) {
	// Setup temp directory and save a test tool
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	// Create a test tool
	testTool := &SavedToolDefinition{
		Name:        "load_test_tool",
		Description: "A tool for load testing",
		InputSchema: map[string]interface{}{"type": "object"},
		Code:        "result = 'loaded successfully'",
	}
	
	if err := SaveTool(testTool); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		name     string
		toolName string
		wantErr  bool
	}{
		{"existing tool", "load_test_tool", false},
		{"non-existent tool", "does_not_exist", true},
		{"empty name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := LoadTool(tt.toolName)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if tool.Name != testTool.Name {
					t.Errorf("LoadTool() name = %q, want %q", tool.Name, testTool.Name)
				}
				if tool.Description != testTool.Description {
					t.Errorf("LoadTool() description = %q, want %q", tool.Description, testTool.Description)
				}
				if tool.Code != testTool.Code {
					t.Errorf("LoadTool() code = %q, want %q", tool.Code, testTool.Code)
				}
			}
		})
	}
}

func TestListTools(t *testing.T) {
	// Setup temp directory
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	t.Run("empty directory", func(t *testing.T) {
		tools, err := ListTools()
		if err != nil {
			t.Errorf("ListTools() error = %v", err)
			return
		}
		if len(tools) != 0 {
			t.Errorf("ListTools() length = %d, want 0", len(tools))
		}
	})

	// Create some test tools
	testTools := []*SavedToolDefinition{
		{
			Name:        "tool_one",
			Description: "First tool",
			Code:        "result = 1",
		},
		{
			Name:        "tool_two", 
			Description: "Second tool",
			Code:        "result = 2",
		},
		{
			Name:        "tool_three",
			Description: "Third tool",
			Code:        "result = 3",
		},
	}

	for _, tool := range testTools {
		if err := SaveTool(tool); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}

	// Create a non-JSON file to test filtering
	toolsDir, _ := GetToolsDirectory()
	os.WriteFile(filepath.Join(toolsDir, "not_a_tool.txt"), []byte("ignore me"), 0644)
	
	// Create a malformed JSON file
	os.WriteFile(filepath.Join(toolsDir, "malformed.json"), []byte("invalid json"), 0644)

	t.Run("with tools", func(t *testing.T) {
		tools, err := ListTools()
		if err != nil {
			t.Errorf("ListTools() error = %v", err)
			return
		}
		
		if len(tools) != 3 {
			t.Errorf("ListTools() length = %d, want 3", len(tools))
			return
		}
		
		// Create a map for easier verification
		toolMap := make(map[string]*SavedToolDefinition)
		for _, tool := range tools {
			toolMap[tool.Name] = tool
		}
		
		for _, expectedTool := range testTools {
			foundTool, exists := toolMap[expectedTool.Name]
			if !exists {
				t.Errorf("ListTools() missing tool: %s", expectedTool.Name)
				continue
			}
			
			if foundTool.Description != expectedTool.Description {
				t.Errorf("ListTools() tool %s description = %q, want %q", 
					expectedTool.Name, foundTool.Description, expectedTool.Description)
			}
			if foundTool.Code != expectedTool.Code {
				t.Errorf("ListTools() tool %s code = %q, want %q",
					expectedTool.Name, foundTool.Code, expectedTool.Code)
			}
		}
	})
}

func TestDeleteTool(t *testing.T) {
	// Setup temp directory
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	// Create a test tool
	testTool := &SavedToolDefinition{
		Name:        "delete_test_tool",
		Description: "A tool for delete testing",
		Code:        "result = 'will be deleted'",
	}
	
	if err := SaveTool(testTool); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		name     string
		toolName string
		wantErr  bool
	}{
		{"existing tool", "delete_test_tool", false},
		{"non-existent tool", "does_not_exist", true},
		{"empty name", "", true},
		{"invalid name", "invalid/name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteTool(tt.toolName)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.toolName == "delete_test_tool" {
				// Verify file was deleted
				toolsDir, _ := GetToolsDirectory()
				filename := filepath.Join(toolsDir, tt.toolName+".json")
				if _, err := os.Stat(filename); !os.IsNotExist(err) {
					t.Errorf("DeleteTool() did not delete file: %s", filename)
				}
				
				// Verify tool is no longer loadable
				if _, err := LoadTool(tt.toolName); err == nil {
					t.Errorf("DeleteTool() tool still loadable after deletion")
				}
			}
		})
	}
}

func TestSaveLoadDeleteWorkflow(t *testing.T) {
	// Setup temp directory
	tempDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tempDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	// Create a complex tool
	originalTool := &SavedToolDefinition{
		Name:        "workflow_test",
		Description: "A tool for testing the complete workflow",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "The message to process",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of times to repeat",
				},
			},
			"required": []string{"message"},
		},
		Code: `message = params.get("message", "Hello")
count = params.get("count", 1)
result = [message] * count`,
	}

	// 1. Save the tool
	if err := SaveTool(originalTool); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 2. Load the tool
	loadedTool, err := LoadTool(originalTool.Name)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded tool matches original
	if loadedTool.Name != originalTool.Name {
		t.Errorf("Loaded tool name = %q, want %q", loadedTool.Name, originalTool.Name)
	}
	if loadedTool.Description != originalTool.Description {
		t.Errorf("Loaded tool description = %q, want %q", loadedTool.Description, originalTool.Description)
	}
	if loadedTool.Code != originalTool.Code {
		t.Errorf("Loaded tool code = %q, want %q", loadedTool.Code, originalTool.Code)
	}

	// 3. Verify it appears in listing
	tools, err := ListTools()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	found := false
	for _, tool := range tools {
		if tool.Name == originalTool.Name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tool not found in listing")
	}

	// 4. Update the tool (save with same name)
	updatedTool := *originalTool
	updatedTool.Description = "Updated description"
	updatedTool.Code = "result = 'updated'"

	if err := SaveTool(&updatedTool); err != nil {
		t.Fatalf("Update save failed: %v", err)
	}

	// Verify update
	reloadedTool, err := LoadTool(originalTool.Name)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if reloadedTool.Description != updatedTool.Description {
		t.Errorf("Updated tool description = %q, want %q", reloadedTool.Description, updatedTool.Description)
	}
	if reloadedTool.Code != updatedTool.Code {
		t.Errorf("Updated tool code = %q, want %q", reloadedTool.Code, updatedTool.Code)
	}

	// 5. Delete the tool
	if err := DeleteTool(originalTool.Name); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	if _, err := LoadTool(originalTool.Name); err == nil {
		t.Errorf("Tool still loadable after deletion")
	}

	// Verify it no longer appears in listing
	tools, err = ListTools()
	if err != nil {
		t.Fatalf("List after delete failed: %v", err)
	}

	for _, tool := range tools {
		if tool.Name == originalTool.Name {
			t.Errorf("Tool still appears in listing after deletion")
		}
	}
}