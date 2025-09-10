package starlark

import (
	"testing"

	"go.starlark.net/starlark"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MockProxyManager for testing
type MockProxyManager struct {
	tools map[string][]*mcp.Tool
	calls []MockCall
}

type MockCall struct {
	ServerName string
	ToolName   string
	Arguments  map[string]interface{}
}

func NewMockProxyManager() *MockProxyManager {
	return &MockProxyManager{
		tools: make(map[string][]*mcp.Tool),
		calls: make([]MockCall, 0),
	}
}

func (m *MockProxyManager) AddServer(serverName string, tools []*mcp.Tool) {
	m.tools[serverName] = tools
}

func (m *MockProxyManager) GetAllTools() map[string][]*mcp.Tool {
	return m.tools
}

func (m *MockProxyManager) CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	m.calls = append(m.calls, MockCall{
		ServerName: serverName,
		ToolName:   toolName,
		Arguments:  arguments,
	})

	// Mock response
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "mock response"},
		},
		StructuredContent: map[string]interface{}{
			"result": "mock response",
			"tool":   toolName,
		},
	}
	return result, nil
}

func TestCreateServerNamespaces(t *testing.T) {
	// Test with nil proxy manager
	namespaces := CreateServerNamespaces(nil)
	if namespaces != nil {
		t.Error("Expected nil namespaces for nil proxy manager")
	}

	// Test with mock proxy manager
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("testserver", []*mcp.Tool{
		{Name: "tool1", Description: "Test tool 1"},
		{Name: "tool2", Description: "Test tool 2"},
	})

	namespaces = CreateServerNamespaces(mockProxy)
	if len(namespaces) != 1 {
		t.Errorf("Expected 1 namespace, got %d", len(namespaces))
	}

	serverNS, exists := namespaces["testserver"]
	if !exists {
		t.Error("Expected testserver namespace to exist")
	}

	if serverNS.Type() != "server_namespace" {
		t.Errorf("Expected type 'server_namespace', got %s", serverNS.Type())
	}
}

func TestServerNamespaceAttrs(t *testing.T) {
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("testserver", []*mcp.Tool{
		{Name: "echo", Description: "Echo tool"},
		{Name: "ping", Description: "Ping tool"},
	})

	namespaces := CreateServerNamespaces(mockProxy)
	serverNS := namespaces["testserver"].(*ServerNamespace)

	// Test valid tool access
	echoTool, err := serverNS.Attr("echo")
	if err != nil {
		t.Errorf("Expected to find echo tool, got error: %v", err)
	}

	if echoTool.Type() != "tool_function" {
		t.Errorf("Expected type 'tool_function', got %s", echoTool.Type())
	}

	// Test invalid tool access
	_, err = serverNS.Attr("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent tool")
	}

	// Test AttrNames
	attrNames := serverNS.AttrNames()
	if len(attrNames) != 2 {
		t.Errorf("Expected 2 attribute names, got %d", len(attrNames))
	}
}

func TestToolFunctionCall(t *testing.T) {
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("testserver", []*mcp.Tool{
		{Name: "echo", Description: "Echo tool"},
	})

	namespaces := CreateServerNamespaces(mockProxy)
	serverNS := namespaces["testserver"].(*ServerNamespace)
	echoTool, _ := serverNS.Attr("echo")
	toolFunc := echoTool.(*ToolFunction)

	// Test function call with dict argument
	thread := &starlark.Thread{Name: "test"}
	testDict := starlark.NewDict(1)
	testDict.SetKey(starlark.String("message"), starlark.String("hello"))
	args := starlark.Tuple{testDict}

	result, err := toolFunc.CallInternal(thread, args, nil)
	if err != nil {
		t.Errorf("Tool call failed: %v", err)
	}

	// Verify the call was made
	if len(mockProxy.calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(mockProxy.calls))
	}

	call := mockProxy.calls[0]
	if call.ServerName != "testserver" || call.ToolName != "echo" {
		t.Errorf("Unexpected call: %+v", call)
	}

	if call.Arguments["message"] != "hello" {
		t.Errorf("Expected message='hello', got %v", call.Arguments["message"])
	}

	// Check result structure
	resultDict, ok := result.(*starlark.Dict)
	if !ok {
		t.Error("Expected result to be a dict")
	}

	// Check that result has expected keys
	contentVal, found, _ := resultDict.Get(starlark.String("content"))
	if !found {
		t.Error("Expected 'content' key in result")
	}

	structuredVal, found, _ := resultDict.Get(starlark.String("structured"))
	if !found {
		t.Error("Expected 'structured' key in result")
	}

	_ = contentVal
	_ = structuredVal
}

func TestToolFunctionCallWithKeywords(t *testing.T) {
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("testserver", []*mcp.Tool{
		{Name: "echo", Description: "Echo tool"},
	})

	namespaces := CreateServerNamespaces(mockProxy)
	serverNS := namespaces["testserver"].(*ServerNamespace)
	echoTool, _ := serverNS.Attr("echo")
	toolFunc := echoTool.(*ToolFunction)

	// Test function call with keyword arguments
	thread := &starlark.Thread{Name: "test"}
	kwargs := []starlark.Tuple{
		{starlark.String("message"), starlark.String("hello")},
		{starlark.String("count"), starlark.MakeInt(3)},
	}

	result, err := toolFunc.CallInternal(thread, starlark.Tuple{}, kwargs)
	if err != nil {
		t.Errorf("Tool call with kwargs failed: %v", err)
	}

	// Verify the call was made with correct arguments
	if len(mockProxy.calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(mockProxy.calls))
	}

	call := mockProxy.calls[0]
	if call.Arguments["message"] != "hello" {
		t.Errorf("Expected message='hello', got %v", call.Arguments["message"])
	}
	if call.Arguments["count"] != int64(3) {
		t.Errorf("Expected count=3, got %v", call.Arguments["count"])
	}

	_ = result
}

func TestToolFunctionCallErrors(t *testing.T) {
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("testserver", []*mcp.Tool{
		{Name: "echo", Description: "Echo tool"},
	})

	namespaces := CreateServerNamespaces(mockProxy)
	serverNS := namespaces["testserver"].(*ServerNamespace)
	echoTool, _ := serverNS.Attr("echo")
	toolFunc := echoTool.(*ToolFunction)

	thread := &starlark.Thread{Name: "test"}

	// Test with invalid argument type
	args := starlark.Tuple{starlark.String("not a dict")}
	_, err := toolFunc.CallInternal(thread, args, nil)
	if err == nil {
		t.Error("Expected error for non-dict argument")
	}

	// Test with too many arguments
	args = starlark.Tuple{starlark.NewDict(0), starlark.NewDict(0)}
	_, err = toolFunc.CallInternal(thread, args, nil)
	if err == nil {
		t.Error("Expected error for too many arguments")
	}
}