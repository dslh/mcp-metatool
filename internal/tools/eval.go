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
	RegisterEvalStarlarkWithProxy(server, nil)
}

// RegisterEvalStarlarkWithProxy registers the eval_starlark tool with proxy support
func RegisterEvalStarlarkWithProxy(server *mcp.Server, proxyManager ProxyManager) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "eval_starlark",
		Description: "Execute Starlark code and return the result",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EvalStarlarkArgs) (*mcp.CallToolResult, any, error) {
		return handleEvalStarlarkWithProxy(ctx, req, args, proxyManager)
	})
}

func handleEvalStarlark(ctx context.Context, req *mcp.CallToolRequest, args EvalStarlarkArgs) (*mcp.CallToolResult, any, error) {
	return handleEvalStarlarkWithProxy(ctx, req, args, nil)
}

func handleEvalStarlarkWithProxy(ctx context.Context, req *mcp.CallToolRequest, args EvalStarlarkArgs, proxyManager ProxyManager) (*mcp.CallToolResult, any, error) {
	// Cast proxyManager to starlark.ProxyManager interface
	var starlarkProxy starlark.ProxyManager
	if proxyManager != nil {
		starlarkProxy = proxyManager
	}
	
	result, err := starlark.ExecuteWithProxy(args.Code, args.Params, starlarkProxy)
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