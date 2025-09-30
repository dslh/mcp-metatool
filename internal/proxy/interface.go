package proxy

import "github.com/modelcontextprotocol/go-sdk/mcp"

// ProxyManager defines the interface for accessing upstream MCP servers
// This is the canonical definition used throughout the codebase
type ProxyManager interface {
	// GetAllTools returns all discovered tools from all connected servers
	GetAllTools() map[string][]*mcp.Tool

	// CallTool invokes a tool on the specified upstream server
	CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}