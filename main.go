package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/persistence"
	"github.com/dslh/mcp-metatool/internal/starlark"
	"github.com/dslh/mcp-metatool/internal/tools"
	"github.com/dslh/mcp-metatool/internal/types"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-metatool",
		Version: "0.1.0",
	}, nil)

	// Register built-in tools
	tools.RegisterEvalStarlark(server)
	tools.RegisterSaveTool(server)

	// Load and register saved tools
	if err := registerSavedTools(server); err != nil {
		log.Printf("Warning: failed to load saved tools: %v", err)
	}

	log.Printf("Starting MCP metatool server...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// registerSavedTools loads all saved tools and registers them as MCP tools
func registerSavedTools(server *mcp.Server) error {
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