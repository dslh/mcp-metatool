package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/starlark"
)

// EvalStarlarkArgs defines the arguments for the eval_starlark tool
type EvalStarlarkArgs struct {
	Code   string                 `json:"code" jsonschema:"the Starlark code to execute"`
	Params map[string]interface{} `json:"params,omitempty" jsonschema:"optional parameters to make available in the execution environment"`
}

// RegisterEvalStarlark registers the eval_starlark tool with the MCP server
func RegisterEvalStarlark(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "eval_starlark",
		Description: "Execute Starlark code and return the result",
	}, handleEvalStarlark)
}

func handleEvalStarlark(ctx context.Context, req *mcp.CallToolRequest, args EvalStarlarkArgs) (*mcp.CallToolResult, any, error) {
	result, err := starlark.Execute(args.Code, args.Params)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Execution failed: %v", err)},
			},
		}, nil, nil
	}

	// Format the result for display
	if result.Error != "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Starlark Error: %s", result.Error)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Result: %v", result.Result)},
		},
	}, result, nil
}