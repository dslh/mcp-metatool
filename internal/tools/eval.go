package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/starlark"
)

// EvalStarlarkArgs defines the arguments for the eval_starlark tool
type EvalStarlarkArgs struct {
	Code   string                 `json:"code" jsonschema:"the Starlark code to execute"`
	Params map[string]interface{} `json:"params,omitempty" jsonschema:"optional parameters to make available in the execution environment"`
}

// RegisterEvalStarlark registers the eval_starlark tool with the MCP server
// The proxyManager parameter is optional; pass nil to register without proxy support
func RegisterEvalStarlark(server *mcp.Server, proxyManager ProxyManager) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "eval_starlark",
		Description: "Execute Starlark code and return the result",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EvalStarlarkArgs) (*mcp.CallToolResult, any, error) {
		return handleEvalStarlark(ctx, req, args, proxyManager)
	})
}

func handleEvalStarlark(ctx context.Context, req *mcp.CallToolRequest, args EvalStarlarkArgs, proxyManager ProxyManager) (*mcp.CallToolResult, any, error) {
	// Cast proxyManager to starlark.ProxyManager interface
	var starlarkProxy starlark.ProxyManager
	if proxyManager != nil {
		starlarkProxy = proxyManager
	}

	result, err := starlark.ExecuteWithProxy(args.Code, args.Params, starlarkProxy)
	if err != nil {
		return ErrorResponse("Execution failed: %v", err), nil, nil
	}

	// Format the result for display
	if result.Error != "" {
		return ErrorResponse("Starlark Error: %s", result.Error), nil, nil
	}

	return SuccessResponse("Result: %v", result.Result), result, nil
}