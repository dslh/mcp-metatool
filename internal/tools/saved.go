package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/persistence"
	"github.com/dslh/mcp-metatool/internal/starlark"
	"github.com/dslh/mcp-metatool/internal/types"
	"github.com/dslh/mcp-metatool/internal/validation"
)

// RegisterSavedTools loads all saved tools and registers them as MCP tools
func RegisterSavedTools(server *mcp.Server) error {
	return RegisterSavedToolsWithProxy(server, nil)
}

// RegisterSavedToolsWithProxy loads all saved tools and registers them with proxy support
func RegisterSavedToolsWithProxy(server *mcp.Server, proxyManager ProxyManager) error {
	savedTools, err := persistence.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list saved tools: %w", err)
	}

	for _, tool := range savedTools {
		// Create a closure to capture the tool definition and proxy manager
		toolDef := tool
		capturedProxy := proxyManager
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolDef.Name,
			Description: toolDef.Description,
		}, func(ctx context.Context, req *mcp.CallToolRequest, args types.SavedToolParams) (*mcp.CallToolResult, any, error) {
			return handleSavedToolWithProxy(toolDef, args, capturedProxy)
		})
		log.Printf("Registered saved tool: %s", tool.Name)
	}

	return nil
}

// handleSavedTool executes a saved tool by running its Starlark code
func handleSavedTool(tool *persistence.SavedToolDefinition, args types.SavedToolParams) (*mcp.CallToolResult, any, error) {
	return handleSavedToolWithProxy(tool, args, nil)
}

// handleSavedToolWithProxy executes a saved tool with proxy manager support
func handleSavedToolWithProxy(tool *persistence.SavedToolDefinition, args types.SavedToolParams, proxyManager ProxyManager) (*mcp.CallToolResult, any, error) {
	// Validate parameters against the tool's input schema
	if err := validation.ValidateParams(tool.InputSchema, map[string]interface{}(args)); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: validation.FormatValidationError(err)},
			},
		}, nil, nil
	}

	// Cast proxyManager to starlark.ProxyManager interface
	var starlarkProxy starlark.ProxyManager
	if proxyManager != nil {
		starlarkProxy = proxyManager
	}

	// Execute the tool's Starlark code with the provided arguments and proxy manager
	result, err := starlark.ExecuteWithProxy(tool.Code, args, starlarkProxy)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Tool execution failed: %v", err)},
			},
		}, nil, nil
	}

	// Handle execution errors
	if result.Error != "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Tool error: %s", result.Error)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Result: %v", result.Result)},
		},
	}, result, nil
}