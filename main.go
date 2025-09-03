package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.starlark.net/starlark"
)

type StarLarkResult struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
	Logs   []string    `json:"logs,omitempty"`
}

func filterUserGlobals(modGlobals, predeclared starlark.StringDict) starlark.StringDict {
	filtered := make(starlark.StringDict)
	for name, value := range modGlobals {
		// Skip predeclared variables (like 'params')
		if _, isPredeclared := predeclared[name]; isPredeclared {
			continue
		}
		// Skip built-in functions and types
		if isBuiltinName(name) {
			continue
		}
		filtered[name] = value
	}
	return filtered
}

func isBuiltinName(name string) bool {
	// Common Starlark built-ins to exclude
	builtins := []string{
		"True", "False", "None",
		"bool", "dict", "enumerate", "float", "getattr", "hasattr",
		"int", "len", "list", "max", "min", "print", "range",
		"repr", "reversed", "sorted", "str", "tuple", "type", "zip",
	}
	for _, builtin := range builtins {
		if name == builtin {
			return true
		}
	}
	return false
}

func executeStarlark(code string, params map[string]interface{}) (*StarLarkResult, error) {
	thread := &starlark.Thread{Name: "eval_starlark"}
	globals := starlark.StringDict{}

	// Convert params to Starlark values if provided
	if params != nil {
		paramsDict := starlark.NewDict(len(params))
		for k, v := range params {
			val, err := goToStarlarkValue(v)
			if err != nil {
				return &StarLarkResult{Error: fmt.Sprintf("Parameter conversion error: %v", err)}, nil
			}
			paramsDict.SetKey(starlark.String(k), val)
		}
		globals["params"] = paramsDict
	}

	// Execute the Starlark code
	var result starlark.Value
	var err error

	// Try as expression first, then as statements
	if strings.Contains(code, "\n") || strings.Contains(code, "return") {
		// Multi-line or contains return - execute as program
		modGlobals, execErr := starlark.ExecFile(thread, "<eval>", code, globals)
		if execErr != nil {
			return &StarLarkResult{Error: fmt.Sprintf("Execution error: %v", execErr)}, nil
		}
		// Look for a 'result' variable first, then fallback to all globals
		if resultVal, ok := modGlobals["result"]; ok {
			result = resultVal
		} else {
			// No explicit result variable - return filtered globals as a dict
			filteredGlobals := filterUserGlobals(modGlobals, globals)
			if len(filteredGlobals) == 0 {
				// No user variables - return None
				result = starlark.None
			} else {
				// Convert filtered globals to Starlark dict
				globalsDict := starlark.NewDict(len(filteredGlobals))
				for k, v := range filteredGlobals {
					globalsDict.SetKey(starlark.String(k), v)
				}
				result = globalsDict
			}
		}
	} else {
		// Single expression - evaluate directly
		result, err = starlark.Eval(thread, "<eval>", code, globals)
		if err != nil {
			return &StarLarkResult{Error: fmt.Sprintf("Evaluation error: %v", err)}, nil
		}
	}

	// Convert result back to Go value
	goResult, err := starlarkToGoValue(result)
	if err != nil {
		return &StarLarkResult{Error: fmt.Sprintf("Result conversion error: %v", err)}, nil
	}

	return &StarLarkResult{Result: goResult}, nil
}

func goToStarlarkValue(v interface{}) (starlark.Value, error) {
	switch val := v.(type) {
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(val), nil
	case int:
		return starlark.MakeInt(val), nil
	case int64:
		return starlark.MakeInt64(val), nil
	case float64:
		return starlark.Float(val), nil
	case string:
		return starlark.String(val), nil
	case []interface{}:
		list := starlark.NewList(make([]starlark.Value, len(val)))
		for i, item := range val {
			starVal, err := goToStarlarkValue(item)
			if err != nil {
				return nil, err
			}
			list.SetIndex(i, starVal)
		}
		return list, nil
	case map[string]interface{}:
		dict := starlark.NewDict(len(val))
		for k, item := range val {
			starVal, err := goToStarlarkValue(item)
			if err != nil {
				return nil, err
			}
			dict.SetKey(starlark.String(k), starVal)
		}
		return dict, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

func starlarkToGoValue(v starlark.Value) (interface{}, error) {
	switch val := v.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.Bool:
		return bool(val), nil
	case starlark.Int:
		if i, ok := val.Int64(); ok {
			return i, nil
		}
		return val.String(), nil // Large integer as string
	case starlark.Float:
		return float64(val), nil
	case starlark.String:
		return string(val), nil
	case *starlark.List:
		result := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			item, err := starlarkToGoValue(val.Index(i))
			if err != nil {
				return nil, err
			}
			result[i] = item
		}
		return result, nil
	case *starlark.Dict:
		result := make(map[string]interface{})
		for _, k := range val.Keys() {
			key, ok := k.(starlark.String)
			if !ok {
				continue // Skip non-string keys
			}
			v, _, _ := val.Get(k)
			goVal, err := starlarkToGoValue(v)
			if err != nil {
				return nil, err
			}
			result[string(key)] = goVal
		}
		return result, nil
	default:
		return val.String(), nil // Fallback to string representation
	}
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-metatool",
		Version: "0.1.0",
	}, nil)

	type HelloWorldArgs struct {
		Name string `json:"name" jsonschema:"the name of the person to greet"`
	}

	// Hello World Tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hello_world",
		Description: "A simple hello world tool that greets a person by name",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HelloWorldArgs) (*mcp.CallToolResult, any, error) {
		greeting := "Hello, " + args.Name + "!"
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: greeting},
			},
		}, nil, nil
	})

	// Eval Starlark Tool
	type EvalStarlarkArgs struct {
		Code   string                 `json:"code" jsonschema:"the Starlark code to execute"`
		Params map[string]interface{} `json:"params,omitempty" jsonschema:"optional parameters to make available in the execution environment"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "eval_starlark",
		Description: "Execute Starlark code and return the result",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EvalStarlarkArgs) (*mcp.CallToolResult, any, error) {
		result, err := executeStarlark(args.Code, args.Params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Execution failed: %v", err)},
				},
			}, nil, nil
		}

		// Format the result as JSON for display
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
	})

	log.Printf("Starting MCP metatool server...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}