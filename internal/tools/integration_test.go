package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/dslh/mcp-metatool/internal/persistence"
)

// MockProxyManager for testing tool integration
type mockProxyManager struct {
	tools map[string][]*mcp.Tool
	calls []mockCall
}

type mockCall struct {
	ServerName string
	ToolName   string
	Arguments  map[string]interface{}
}

func newMockProxyManager() *mockProxyManager {
	return &mockProxyManager{
		tools: make(map[string][]*mcp.Tool),
		calls: make([]mockCall, 0),
	}
}

func (m *mockProxyManager) AddServer(serverName string, tools []*mcp.Tool) {
	m.tools[serverName] = tools
}

func (m *mockProxyManager) GetAllTools() map[string][]*mcp.Tool {
	return m.tools
}

func (m *mockProxyManager) CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	m.calls = append(m.calls, mockCall{
		ServerName: serverName,
		ToolName:   toolName,
		Arguments:  arguments,
	})

	// Mock response
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "mock response from " + serverName + "." + toolName},
		},
		StructuredContent: map[string]interface{}{
			"result":     "mock response",
			"server":     serverName,
			"tool":       toolName,
			"arguments":  arguments,
		},
	}
	return result, nil
}

func TestEvalStarlarkWithProxyIntegration(t *testing.T) {
	// Create mock proxy manager
	mockProxy := newMockProxyManager()
	mockProxy.AddServer("echo", []*mcp.Tool{
		{Name: "echo", Description: "Echo tool"},
	})
	mockProxy.AddServer("test", []*mcp.Tool{
		{Name: "greet", Description: "Greeting tool"},
	})

	// Create and register eval_starlark tool with proxy
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	RegisterEvalStarlark(server, mockProxy)

	// Test calling the tool with a Starlark script that uses proxied tools
	ctx := context.Background()
	
	code := `
# Test calling multiple proxied tools
echo_result = echo.echo({"message": "hello"})
greet_result = test.greet({"name": "Alice"})

result = {
    "echo_response": echo_result["structured"]["result"],
    "greet_response": greet_result["structured"]["result"],
    "servers_used": ["echo", "test"]
}
result
`

	// Call the tool handler directly
	args := EvalStarlarkArgs{
		Code: code,
	}
	
	req := &mcp.CallToolRequest{}

	result, _, err := handleEvalStarlark(ctx, req, args, mockProxy)
	if err != nil {
		t.Errorf("handleEvalStarlarkWithProxy failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify that both tools were called
	if len(mockProxy.calls) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(mockProxy.calls))
	}

	// Check the calls
	expectedCalls := []struct {
		server string
		tool   string
	}{
		{"echo", "echo"},
		{"test", "greet"},
	}

	for i, expected := range expectedCalls {
		if i >= len(mockProxy.calls) {
			t.Errorf("Missing call %d: expected %s.%s", i, expected.server, expected.tool)
			continue
		}
		call := mockProxy.calls[i]
		if call.ServerName != expected.server || call.ToolName != expected.tool {
			t.Errorf("Call %d: expected %s.%s, got %s.%s", i, expected.server, expected.tool, call.ServerName, call.ToolName)
		}
	}

	// Verify the arguments
	if len(mockProxy.calls) > 0 {
		echoCall := mockProxy.calls[0]
		if echoCall.Arguments["message"] != "hello" {
			t.Errorf("Echo call: expected message='hello', got %v", echoCall.Arguments["message"])
		}
	}

	if len(mockProxy.calls) > 1 {
		greetCall := mockProxy.calls[1]
		if greetCall.Arguments["name"] != "Alice" {
			t.Errorf("Greet call: expected name='Alice', got %v", greetCall.Arguments["name"])
		}
	}
}

func TestEvalStarlarkWithoutProxy(t *testing.T) {
	// Test that eval_starlark still works without proxy manager
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	RegisterEvalStarlark(server, nil)

	ctx := context.Background()
	code := "2 + 3 * 4"

	args := EvalStarlarkArgs{
		Code: code,
	}

	req := &mcp.CallToolRequest{}

	result, _, err := handleEvalStarlark(ctx, req, args, nil)
	if err != nil {
		t.Errorf("handleEvalStarlark failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check that we get the expected result
	if len(result.Content) == 0 {
		t.Fatal("Expected content in result")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if textContent.Text != "Result: 14" {
		t.Errorf("Expected 'Result: 14', got %s", textContent.Text)
	}
}

func TestSavedToolsWithProxyIntegration(t *testing.T) {
	// Test that saved tools can also use proxy manager
	mockProxy := newMockProxyManager()
	mockProxy.AddServer("api", []*mcp.Tool{
		{Name: "fetch", Description: "Fetch data"},
	})

	// Test the saved tool handler directly
	toolDef := &persistence.SavedToolDefinition{
		Name:        "test_composite",
		Description: "Test composite tool",
		Code: `
# Test composite tool logic
api_result = api.fetch({"url": params["url"]})
result = {
    "status": "success",
    "data": api_result["structured"]["result"],
    "url": params["url"]
}
result
`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"url"},
		},
	}

	args := map[string]interface{}{
		"url": "https://api.example.com/data",
	}

	result, _, err := handleSavedTool(toolDef, args, mockProxy)
	if err != nil {
		t.Errorf("handleSavedToolWithProxy failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify that the API tool was called
	if len(mockProxy.calls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(mockProxy.calls))
	}

	if len(mockProxy.calls) > 0 {
		call := mockProxy.calls[0]
		if call.ServerName != "api" || call.ToolName != "fetch" {
			t.Errorf("Expected api.fetch call, got %s.%s", call.ServerName, call.ToolName)
		}
		if call.Arguments["url"] != "https://api.example.com/data" {
			t.Errorf("Expected url parameter, got %v", call.Arguments["url"])
		}
	}
}

func TestProxyManagerInterface(t *testing.T) {
	// Test that our mock implements the interface correctly
	var _ ProxyManager = (*mockProxyManager)(nil)

	mock := newMockProxyManager()
	tools := mock.GetAllTools()
	if tools == nil {
		t.Error("GetAllTools should not return nil")
	}

	mock.AddServer("test", []*mcp.Tool{
		{Name: "tool1", Description: "Test tool 1"},
	})

	tools = mock.GetAllTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 server, got %d", len(tools))
	}

	if _, exists := tools["test"]; !exists {
		t.Error("Expected 'test' server to exist")
	}

	// Test tool call
	result, err := mock.CallTool("test", "tool1", map[string]interface{}{"param": "value"})
	if err != nil {
		t.Errorf("CallTool failed: %v", err)
	}
	if result == nil {
		t.Error("Expected result from CallTool")
	}

	if len(mock.calls) != 1 {
		t.Errorf("Expected 1 call recorded, got %d", len(mock.calls))
	}
}