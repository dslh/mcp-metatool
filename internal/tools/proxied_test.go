package tools

import (
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/config"
)

// MockProxyManager implements a minimal proxy manager for testing
type MockProxyManager struct {
	tools map[string][]*mcp.Tool
	callResults map[string]*mcp.CallToolResult
}

func NewMockProxyManager() *MockProxyManager {
	return &MockProxyManager{
		tools: make(map[string][]*mcp.Tool),
		callResults: make(map[string]*mcp.CallToolResult),
	}
}

func (m *MockProxyManager) GetAllTools() map[string][]*mcp.Tool {
	return m.tools
}

func (m *MockProxyManager) CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	key := serverName + ":" + toolName
	if result, exists := m.callResults[key]; exists {
		return result, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Mock result"},
		},
	}, nil
}

func (m *MockProxyManager) AddMockTool(serverName string, tool *mcp.Tool) {
	if m.tools[serverName] == nil {
		m.tools[serverName] = []*mcp.Tool{}
	}
	m.tools[serverName] = append(m.tools[serverName], tool)
}

func (m *MockProxyManager) SetMockResult(serverName, toolName string, result *mcp.CallToolResult) {
	key := serverName + ":" + toolName
	m.callResults[key] = result
}

func TestRegisterProxiedTools(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		config  *config.Config
		tools   map[string][]*mcp.Tool
		wantErr bool
	}{
		{
			name:   "register tools from visible servers",
			envVar: "",
			config: &config.Config{
				MCPServers: map[string]config.MCPServerConfig{
					"github": {Command: "test", Hidden: false},
					"slack":  {Command: "test", Hidden: false},
				},
			},
			tools: map[string][]*mcp.Tool{
				"github": {
					{Name: "create_issue", Description: "Create a GitHub issue"},
					{Name: "list_repos", Description: "List repositories"},
				},
				"slack": {
					{Name: "send_message", Description: "Send a Slack message"},
				},
			},
			wantErr: false,
		},
		{
			name:   "skip hidden servers",
			envVar: "",
			config: &config.Config{
				MCPServers: map[string]config.MCPServerConfig{
					"github": {Command: "test", Hidden: false},
					"slack":  {Command: "test", Hidden: true},
				},
			},
			tools: map[string][]*mcp.Tool{
				"github": {
					{Name: "create_issue", Description: "Create a GitHub issue"},
				},
				"slack": {
					{Name: "send_message", Description: "Send a Slack message"},
				},
			},
			wantErr: false,
		},
		{
			name:   "skip all tools when env var is set",
			envVar: "true",
			config: &config.Config{
				MCPServers: map[string]config.MCPServerConfig{
					"github": {Command: "test", Hidden: false},
				},
			},
			tools: map[string][]*mcp.Tool{
				"github": {
					{Name: "create_issue", Description: "Create a GitHub issue"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			os.Unsetenv("MCP_METATOOL_HIDE_PROXIED_TOOLS")
			if tt.envVar != "" {
				os.Setenv("MCP_METATOOL_HIDE_PROXIED_TOOLS", tt.envVar)
				defer os.Unsetenv("MCP_METATOOL_HIDE_PROXIED_TOOLS")
			}

			// Create mock server and proxy manager
			server := mcp.NewServer(&mcp.Implementation{
				Name:    "test-server",
				Version: "1.0.0",
			}, nil)
			
			mockProxy := NewMockProxyManager()
			for serverName, tools := range tt.tools {
				for _, tool := range tools {
					mockProxy.AddMockTool(serverName, tool)
				}
			}

			// Register proxied tools
			err := RegisterProxiedTools(server, mockProxy, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterProxiedTools() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleProxiedTool(t *testing.T) {
	mockProxy := NewMockProxyManager()
	
	// Set up mock result
	expectedResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Test result from GitHub"},
		},
	}
	mockProxy.SetMockResult("github", "create_issue", expectedResult)

	// Test the handler
	args := ProxiedToolArgs{
		"title": "Test Issue",
		"body":  "This is a test issue",
	}

	result, _, err := handleProxiedTool(mockProxy, "github", "create_issue", args)
	if err != nil {
		t.Fatalf("handleProxiedTool failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if textContent.Text != "Test result from GitHub" {
		t.Errorf("Expected 'Test result from GitHub', got '%s'", textContent.Text)
	}
}

func TestHandleProxiedToolWithStructuredContent(t *testing.T) {
	// Test that structured content is properly extracted and no circular reference occurs
	mockProxy := NewMockProxyManager()
	
	// Create a result with both content and structured content
	structuredData := map[string]interface{}{
		"result":       "Hello from test!",
		"command_args": []string{"test", "arg1", "arg2"},
		"timestamp":    "2025-09-10T23:30:00Z",
	}
	
	expectedResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Regular content response"},
		},
		StructuredContent: structuredData,
		IsError:          false,
	}
	mockProxy.SetMockResult("echo", "echo", expectedResult)

	// Test the handler
	args := ProxiedToolArgs{
		"message": "Hello from test!",
	}

	result, structuredContent, err := handleProxiedTool(mockProxy, "echo", "echo", args)
	if err != nil {
		t.Fatalf("handleProxiedTool failed: %v", err)
	}

	// Verify the result is not nil
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify content is preserved
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if textContent.Text != "Regular content response" {
		t.Errorf("Expected 'Regular content response', got '%s'", textContent.Text)
	}

	// Verify structured content is extracted correctly
	if structuredContent == nil {
		t.Fatal("Expected structured content, got nil")
	}

	// Verify structured content is the same as what we set
	structuredMap, ok := structuredContent.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected structured content to be map[string]interface{}, got %T", structuredContent)
	}

	if structuredMap["result"] != "Hello from test!" {
		t.Errorf("Expected result 'Hello from test!', got '%v'", structuredMap["result"])
	}

	// Most importantly: verify that structured content is NOT the same object as result
	// This prevents the circular reference bug
	if structuredContent == result {
		t.Fatal("Structured content should not be the same object as result - this would cause circular reference!")
	}
}

func TestRegisterProxiedToolsWithMissingServerInConfig(t *testing.T) {
	// Test behavior when proxy manager has tools from a server not in config
	config := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{
			"github": {Command: "test", Hidden: false},
		},
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	mockProxy := NewMockProxyManager()
	// Add tools from a server not in config
	mockProxy.AddMockTool("unknown", &mcp.Tool{
		Name:        "unknown_tool",
		Description: "Tool from unknown server",
	})
	mockProxy.AddMockTool("github", &mcp.Tool{
		Name:        "create_issue",
		Description: "Create a GitHub issue",
	})

	err := RegisterProxiedTools(server, mockProxy, config)
	if err != nil {
		t.Fatalf("RegisterProxiedTools failed: %v", err)
	}

	// The function should complete successfully even with unknown servers
}