package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/config"
	"github.com/dslh/mcp-metatool/internal/proxy"
	"github.com/dslh/mcp-metatool/internal/schema"
)

// ProxyManager is an alias for the canonical interface
type ProxyManager = proxy.ProxyManager

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
		// Get server configuration
		serverConfig, exists := cfg.MCPServers[serverName]
		if !exists {
			log.Printf("Warning: No configuration found for server %s, skipping tools", serverName)
			continue
		}

		// Check if this specific server should be hidden
		if serverConfig.Hidden {
			log.Printf("Skipping tools from hidden server: %s", serverName)
			continue
		}

		for _, tool := range tools {
			// Check if this tool should be included based on server configuration
			if !serverConfig.ShouldIncludeTool(tool.Name) {
				log.Printf("Filtered out tool: %s.%s", serverName, tool.Name)
				continue
			}

			// Create a prefixed tool name to avoid conflicts
			prefixedName := fmt.Sprintf("%s__%s", serverName, tool.Name)

			// Create a closure to capture the server and tool names
			capturedServerName := serverName
			capturedToolName := tool.Name

			// Transform the schema to ensure compatibility with draft-2020-12
			transformedSchema := schema.SafeTransform(tool.InputSchema, fmt.Sprintf("tool %s", tool.Name))

			mcp.AddTool(server, &mcp.Tool{
				Name:        prefixedName,
				Description: fmt.Sprintf("[%s] %s", serverName, tool.Description),
				InputSchema: transformedSchema,
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
		return ErrorResponse("Proxied tool call failed: %v", err), nil, nil
	}

	// Return the result from the upstream server
	return result, result.StructuredContent, nil
}