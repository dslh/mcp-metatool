package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/persistence"
	"github.com/dslh/mcp-metatool/internal/types"
)

// RegisterSaveTool registers the save_tool tool with the MCP server
func RegisterSaveTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "save_tool",
		Description: "Create or update a composite tool definition",
	}, handleSaveTool)
}

func handleSaveTool(ctx context.Context, req *mcp.CallToolRequest, args types.SaveToolArgs) (*mcp.CallToolResult, any, error) {
	// Basic validation
	if args.Name == "" {
		return ErrorResponse("Error: tool name is required"), nil, nil
	}

	if args.Description == "" {
		return ErrorResponse("Error: tool description is required"), nil, nil
	}

	if args.Code == "" {
		return ErrorResponse("Error: tool code is required"), nil, nil
	}

	// Create tool definition
	tool := &persistence.SavedToolDefinition{
		Name:        args.Name,
		Description: args.Description,
		InputSchema: args.InputSchema,
		Code:        args.Code,
	}

	// Save to disk
	if err := persistence.SaveTool(tool); err != nil {
		return ErrorResponse("Failed to save tool: %v", err), nil, nil
	}

	return SuccessResponse("Tool '%s' saved successfully", args.Name), tool, nil
}