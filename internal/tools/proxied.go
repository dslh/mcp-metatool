package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/config"
)

// ProxyManager interface allows for easier testing
type ProxyManager interface {
	GetAllTools() map[string][]*mcp.Tool
	CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}

// ProxiedToolArgs represents the arguments for a proxied tool call
type ProxiedToolArgs map[string]interface{}

// RegisterProxiedTools registers all discovered tools from upstream MCP servers
func RegisterProxiedTools(server *mcp.Server, proxyManager ProxyManager, cfg *config.Config) error {
	// Check if proxied tools should be hidden globally
	if config.ShouldHideProxiedTools() {
		log.Printf("Proxied tools are hidden via MCP_METATOOL_HIDE_PROXIED_TOOLS environment variable")
		return nil
	}

	allTools := proxyManager.GetAllTools()
	totalRegistered := 0

	for serverName, tools := range allTools {
		// Check if this specific server should be hidden
		if serverConfig, exists := cfg.MCPServers[serverName]; exists && serverConfig.Hidden {
			log.Printf("Skipping tools from hidden server: %s", serverName)
			continue
		}

		for _, tool := range tools {
			// Create a prefixed tool name to avoid conflicts
			prefixedName := fmt.Sprintf("%s__%s", serverName, tool.Name)
			
			// Create a closure to capture the server and tool names
			capturedServerName := serverName
			capturedToolName := tool.Name
			
			mcp.AddTool(server, &mcp.Tool{
				Name:        prefixedName,
				Description: fmt.Sprintf("[%s] %s", serverName, tool.Description),
				InputSchema: tool.InputSchema,
			}, func(ctx context.Context, req *mcp.CallToolRequest, args ProxiedToolArgs) (*mcp.CallToolResult, any, error) {
				return handleProxiedTool(proxyManager, capturedServerName, capturedToolName, args)
			})
			
			log.Printf("Registered proxied tool: %s -> %s.%s", prefixedName, serverName, tool.Name)
			totalRegistered++
		}
	}

	log.Printf("Successfully registered %d proxied tools from %d servers", totalRegistered, len(allTools))
	return nil
}

// handleProxiedTool forwards a tool call to the appropriate upstream server
func handleProxiedTool(proxyManager ProxyManager, serverName, toolName string, args ProxiedToolArgs) (*mcp.CallToolResult, any, error) {
	// Forward the call to the upstream server
	result, err := proxyManager.CallTool(serverName, toolName, map[string]interface{}(args))
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Proxied tool call failed: %v", err)},
			},
		}, nil, nil
	}

	// Return the result from the upstream server
	return result, result.StructuredContent, nil
}