package starlark

import (
	"fmt"

	"go.starlark.net/starlark"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProxyManager interface for accessing upstream MCP servers
// This is a local definition to avoid circular imports with the tools package
type ProxyManager interface {
	GetAllTools() map[string][]*mcp.Tool
	CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}

// ServerNamespace represents a server's tools as a Starlark object
type ServerNamespace struct {
	serverName   string
	proxyManager ProxyManager
	tools        map[string]*mcp.Tool // toolName -> tool definition
}

// String implements starlark.Value
func (s *ServerNamespace) String() string {
	return fmt.Sprintf("<%s server namespace>", s.serverName)
}

// Type implements starlark.Value
func (s *ServerNamespace) Type() string {
	return "server_namespace"
}

// Freeze implements starlark.Value
func (s *ServerNamespace) Freeze() {}

// Truth implements starlark.Value
func (s *ServerNamespace) Truth() starlark.Bool {
	return len(s.tools) > 0
}

// Hash implements starlark.Value
func (s *ServerNamespace) Hash() (uint32, error) {
	return starlark.String(s.serverName).Hash()
}

// Attr implements starlark.HasAttrs to provide tool access via dot notation
func (s *ServerNamespace) Attr(name string) (starlark.Value, error) {
	tool, exists := s.tools[name]
	if !exists {
		return nil, starlark.NoSuchAttrError(fmt.Sprintf("server '%s' has no tool '%s'", s.serverName, name))
	}
	
	// Return a callable function for this tool
	return &ToolFunction{
		serverName:   s.serverName,
		toolName:     name,
		tool:         tool,
		proxyManager: s.proxyManager,
	}, nil
}

// AttrNames implements starlark.HasAttrs
func (s *ServerNamespace) AttrNames() []string {
	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	return names
}

// ToolFunction represents a callable tool function in Starlark
type ToolFunction struct {
	serverName   string
	toolName     string
	tool         *mcp.Tool
	proxyManager ProxyManager
}

// String implements starlark.Value
func (t *ToolFunction) String() string {
	return fmt.Sprintf("<%s.%s tool function>", t.serverName, t.toolName)
}

// Type implements starlark.Value
func (t *ToolFunction) Type() string {
	return "tool_function"
}

// Freeze implements starlark.Value
func (t *ToolFunction) Freeze() {}

// Truth implements starlark.Value
func (t *ToolFunction) Truth() starlark.Bool {
	return true
}

// Hash implements starlark.Value
func (t *ToolFunction) Hash() (uint32, error) {
	return starlark.String(fmt.Sprintf("%s.%s", t.serverName, t.toolName)).Hash()
}

// Name implements starlark.Callable
func (t *ToolFunction) Name() string {
	return fmt.Sprintf("%s.%s", t.serverName, t.toolName)
}

// CallInternal implements starlark.Callable
func (t *ToolFunction) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// Convert arguments to Go map
	var params map[string]interface{}
	
	if len(args) == 0 && len(kwargs) == 0 {
		params = make(map[string]interface{})
	} else if len(args) == 1 && len(kwargs) == 0 {
		// Single positional argument should be a dict
		if dict, ok := args[0].(*starlark.Dict); ok {
			convertedVal, err := StarlarkToGoValue(dict)
			if err != nil {
				return nil, fmt.Errorf("failed to convert arguments: %v", err)
			}
			// Ensure it's a map
			if paramMap, ok := convertedVal.(map[string]interface{}); ok {
				params = paramMap
			} else {
				return nil, fmt.Errorf("argument must be a dict/map")
			}
		} else {
			return nil, fmt.Errorf("single argument must be a dict")
		}
	} else if len(args) == 0 && len(kwargs) > 0 {
		// Keyword arguments
		params = make(map[string]interface{})
		for _, kw := range kwargs {
			if len(kw) != 2 {
				return nil, fmt.Errorf("invalid keyword argument")
			}
			key, ok := kw[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("keyword argument key must be string")
			}
			value, err := StarlarkToGoValue(kw[1])
			if err != nil {
				return nil, fmt.Errorf("failed to convert keyword argument: %v", err)
			}
			params[string(key)] = value
		}
	} else {
		return nil, fmt.Errorf("tool functions accept either a single dict argument or keyword arguments")
	}
	
	// Call the proxied tool
	result, err := t.proxyManager.CallTool(t.serverName, t.toolName, params)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %v", err)
	}
	
	// Convert result back to Starlark
	// For now, we'll return a simple dict with the content
	resultDict := starlark.NewDict(0)
	
	// Add content as a list
	if len(result.Content) > 0 {
		contentList := starlark.NewList(make([]starlark.Value, len(result.Content)))
		for i, content := range result.Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				contentList.SetIndex(i, starlark.String(textContent.Text))
			} else {
				// For other content types, convert to string
				contentList.SetIndex(i, starlark.String(fmt.Sprintf("%v", content)))
			}
		}
		resultDict.SetKey(starlark.String("content"), contentList)
	}
	
	// Add structured content if available
	if result.StructuredContent != nil {
		structuredVal, err := GoToStarlarkValue(result.StructuredContent)
		if err == nil {
			resultDict.SetKey(starlark.String("structured"), structuredVal)
		}
	}
	
	return resultDict, nil
}

// CreateServerNamespaces creates Starlark server namespace objects from a ProxyManager
func CreateServerNamespaces(proxyManager ProxyManager) starlark.StringDict {
	if proxyManager == nil {
		return nil
	}
	
	allTools := proxyManager.GetAllTools()
	namespaces := make(starlark.StringDict)
	
	for serverName, tools := range allTools {
		toolMap := make(map[string]*mcp.Tool)
		for _, tool := range tools {
			toolMap[tool.Name] = tool
		}
		
		namespace := &ServerNamespace{
			serverName:   serverName,
			proxyManager: proxyManager,
			tools:        toolMap,
		}
		
		namespaces[serverName] = namespace
	}
	
	return namespaces
}