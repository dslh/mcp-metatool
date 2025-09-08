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
	savedTools, err := persistence.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list saved tools: %w", err)
	}

	for _, tool := range savedTools {
		// Create a closure to capture the tool definition
		toolDef := tool
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolDef.Name,
			Description: toolDef.Description,
		}, func(ctx context.Context, req *mcp.CallToolRequest, args types.SavedToolParams) (*mcp.CallToolResult, any, error) {
			return handleSavedTool(toolDef, args)
		})
		log.Printf("Registered saved tool: %s", tool.Name)
	}

	return nil
}

// handleSavedTool executes a saved tool by running its Starlark code
func handleSavedTool(tool *persistence.SavedToolDefinition, args types.SavedToolParams) (*mcp.CallToolResult, any, error) {
	// Validate parameters against the tool's input schema
	if err := validation.ValidateParams(tool.InputSchema, map[string]interface{}(args)); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: validation.FormatValidationError(err)},
			},
		}, nil, nil
	}

	// Execute the tool's Starlark code with the provided arguments
	result, err := starlark.Execute(tool.Code, args)
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