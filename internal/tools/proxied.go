package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/google/jsonschema-go/jsonschema"
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

// transformSchema converts JSON Schema draft-07 to draft-2020-12 for compatibility
func transformSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return nil
	}

	// Create a copy to avoid modifying the original
	transformed := *schema

	// Handle the main compatibility issue: transform draft-07 $schema to draft-2020-12
	if schema.Schema == "http://json-schema.org/draft-07/schema#" ||
		schema.Schema == "http://json-schema.org/draft-07/schema" {
		transformed.Schema = "https://json-schema.org/draft/2020-12/schema"
	}

	// Recursively transform nested schemas in properties
	if schema.Properties != nil {
		transformed.Properties = make(map[string]*jsonschema.Schema)
		for k, v := range schema.Properties {
			transformed.Properties[k] = transformSchema(v)
		}
	}

	// Transform items schema if present
	if schema.Items != nil {
		transformed.Items = transformSchema(schema.Items)
	}

	// Transform additional properties schema if present
	if schema.AdditionalProperties != nil {
		transformed.AdditionalProperties = transformSchema(schema.AdditionalProperties)
	}

	return &transformed
}

// safeTransformSchema safely transforms a schema with error handling
func safeTransformSchema(schema *jsonschema.Schema, toolName string) *jsonschema.Schema {
	var result *jsonschema.Schema
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Warning: Schema transformation failed for tool %s: %v. Proceeding without schema validation.", toolName, r)
			result = nil
		}
	}()

	result = transformSchema(schema)
	return result
}

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
			transformedSchema := safeTransformSchema(tool.InputSchema, tool.Name)

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