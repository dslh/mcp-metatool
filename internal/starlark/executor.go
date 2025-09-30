package starlark

import (
	"fmt"
	"strings"

	"go.starlark.net/lib/json"
	"go.starlark.net/lib/math"
	"go.starlark.net/lib/time"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Result represents the result of executing Starlark code
type Result struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
	Logs   []string    `json:"logs,omitempty"`
}

// Execute runs Starlark code with optional parameters and returns the result
func Execute(code string, params map[string]interface{}) (*Result, error) {
	return ExecuteWithProxy(code, params, nil)
}

// ExecuteWithProxy runs Starlark code with optional parameters and proxy manager access
func ExecuteWithProxy(code string, params map[string]interface{}, proxyManager ProxyManager) (*Result, error) {
	thread := &starlark.Thread{Name: "eval_starlark"}
	
	// Set up predeclared identifiers (built-ins + params)
	predeclared := make(starlark.StringDict)
	for name, value := range starlark.Universe {
		predeclared[name] = value
	}

	// Add standard library modules
	predeclared["time"] = time.Module
	predeclared["math"] = math.Module
	predeclared["json"] = json.Module

	// Convert params to Starlark values if provided
	if params != nil {
		paramsDict := starlark.NewDict(len(params))
		for k, v := range params {
			val, err := GoToStarlarkValue(v)
			if err != nil {
				return &Result{Error: fmt.Sprintf("Parameter conversion error: %v", err)}, nil
			}
			paramsDict.SetKey(starlark.String(k), val)
		}
		predeclared["params"] = paramsDict
	}

	// Add server namespaces if proxy manager is available
	if proxyManager != nil {
		serverNamespaces := CreateServerNamespaces(proxyManager)
		for name, namespace := range serverNamespaces {
			predeclared[name] = namespace
		}
	}

	// Execute the Starlark code
	var result starlark.Value
	var err error

	// Configure Starlark with full language features
	fileOptions := &syntax.FileOptions{
		Set:             true, // Enable set literals and comprehensions
		While:           true, // Enable while loops
		TopLevelControl: true, // Enable for loops and if statements at top level
		GlobalReassign:  true, // Allow reassignment of global variables
		LoadBindsGlobally: true, // Load statements bind globally
	}

	// Execute the code and extract result
	result, err = executeCode(code, fileOptions, thread, predeclared)
	if err != nil {
		return &Result{Error: err.Error()}, nil
	}

	// Convert result back to Go value
	goResult, err := StarlarkToGoValue(result)
	if err != nil {
		return &Result{Error: fmt.Sprintf("Result conversion error: %v", err)}, nil
	}

	return &Result{Result: goResult}, nil
}

// executeCode runs Starlark code and extracts the result
func executeCode(code string, fileOptions *syntax.FileOptions, thread *starlark.Thread, predeclared starlark.StringDict) (starlark.Value, error) {
	// Check if code should be executed as a program or expression
	if isMultiLineCode(code) {
		return executeAsProgram(code, fileOptions, thread, predeclared)
	}
	return executeAsExpression(code, fileOptions, thread, predeclared)
}

// isMultiLineCode determines if code should be executed as a program
func isMultiLineCode(code string) bool {
	return strings.Contains(code, "\n") || strings.Contains(code, "return")
}

// executeAsProgram executes code as a Starlark program and extracts the result
func executeAsProgram(code string, fileOptions *syntax.FileOptions, thread *starlark.Thread, predeclared starlark.StringDict) (starlark.Value, error) {
	modGlobals, err := starlark.ExecFileOptions(fileOptions, thread, "<eval>", code, predeclared)
	if err != nil {
		return nil, fmt.Errorf("Execution error: %v", err)
	}

	// Look for explicit 'result' variable first
	if resultVal, ok := modGlobals["result"]; ok {
		return resultVal, nil
	}

	// No explicit result - return filtered globals
	return extractResultFromGlobals(modGlobals, predeclared), nil
}

// executeAsExpression evaluates code as a single expression
func executeAsExpression(code string, fileOptions *syntax.FileOptions, thread *starlark.Thread, predeclared starlark.StringDict) (starlark.Value, error) {
	result, err := starlark.EvalOptions(fileOptions, thread, "<eval>", code, predeclared)
	if err != nil {
		return nil, fmt.Errorf("Evaluation error: %v", err)
	}
	return result, nil
}

// extractResultFromGlobals filters user-defined globals and returns them as a result
func extractResultFromGlobals(modGlobals, predeclared starlark.StringDict) starlark.Value {
	filteredGlobals := filterUserGlobals(modGlobals, predeclared)

	if len(filteredGlobals) == 0 {
		return starlark.None
	}

	// Convert filtered globals to Starlark dict
	globalsDict := starlark.NewDict(len(filteredGlobals))
	for k, v := range filteredGlobals {
		globalsDict.SetKey(starlark.String(k), v)
	}
	return globalsDict
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
