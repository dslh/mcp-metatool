package tools

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrorResponse creates a standardized error response for tool calls
func ErrorResponse(format string, args ...interface{}) *mcp.CallToolResult {
	message := fmt.Sprintf(format, args...)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
	}
}

// SuccessResponse creates a standardized success response for tool calls
func SuccessResponse(format string, args ...interface{}) *mcp.CallToolResult {
	message := fmt.Sprintf(format, args...)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
	}
}