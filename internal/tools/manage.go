package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/persistence"
	"github.com/dslh/mcp-metatool/internal/types"
)

// ToolSummary represents a summary of a saved tool for list_saved_tools
type ToolSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToolListResponse wraps the tool list in an object structure expected by MCP
type ToolListResponse struct {
	Tools []ToolSummary `json:"tools"`
}

// RegisterListSavedTools registers the list_saved_tools tool with the MCP server
func RegisterListSavedTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_saved_tools",
		Description: "List all saved composite tool definitions",
	}, handleListSavedTools)
}

// RegisterShowSavedTool registers the show_saved_tool tool with the MCP server
func RegisterShowSavedTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "show_saved_tool",
		Description: "Show the complete definition of a saved tool",
	}, handleShowSavedTool)
}

// RegisterDeleteSavedTool registers the delete_saved_tool tool with the MCP server
func RegisterDeleteSavedTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_saved_tool",
		Description: "Delete a saved tool definition",
	}, handleDeleteSavedTool)
}

func handleListSavedTools(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	// Get all saved tools
	tools, err := persistence.ListTools()
	if err != nil {
		return ErrorResponse("Failed to list saved tools: %v", err), nil, nil
	}

	// Convert to summary format
	var summaries []ToolSummary
	for _, tool := range tools {
		summaries = append(summaries, ToolSummary{
			Name:        tool.Name,
			Description: tool.Description,
		})
	}

	// Wrap in object structure
	response := ToolListResponse{Tools: summaries}

	if len(summaries) == 0 {
		return SuccessResponse("No saved tools found"), response, nil
	}

	// Build a readable list of tools
	var toolList []string
	for _, tool := range summaries {
		toolList = append(toolList, fmt.Sprintf("• %s: %s", tool.Name, tool.Description))
	}

	listText := fmt.Sprintf("Found %d saved tool(s):\n\n%s", len(summaries), strings.Join(toolList, "\n"))

	return SuccessResponse(listText), response, nil
}

func handleShowSavedTool(ctx context.Context, req *mcp.CallToolRequest, args types.ShowToolArgs) (*mcp.CallToolResult, any, error) {
	// Validate arguments
	if args.Name == "" {
		return ErrorResponse("Error: tool name is required"), nil, nil
	}

	// Load the tool
	tool, err := persistence.LoadTool(args.Name)
	if err != nil {
		return ErrorResponse("Failed to load tool '%s': %v", args.Name, err), nil, nil
	}

	return SuccessResponse(tool.Code), tool, nil
}

func handleDeleteSavedTool(ctx context.Context, req *mcp.CallToolRequest, args types.DeleteToolArgs) (*mcp.CallToolResult, any, error) {
	// Validate arguments
	if args.Name == "" {
		return ErrorResponse("Error: tool name is required"), nil, nil
	}

	// Delete the tool
	err := persistence.DeleteTool(args.Name)
	if err != nil {
		return ErrorResponse("Failed to delete tool '%s': %v", args.Name, err), nil, nil
	}

	return SuccessResponse("Tool '%s' deleted successfully. Restart server to remove from available tools.", args.Name), map[string]string{"deleted": args.Name}, nil
}