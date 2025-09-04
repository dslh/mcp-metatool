package starlark

import (
	"fmt"
	"strings"

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
	thread := &starlark.Thread{Name: "eval_starlark"}
	
	// Set up predeclared identifiers (built-ins + params)
	predeclared := make(starlark.StringDict)
	for name, value := range starlark.Universe {
		predeclared[name] = value
	}

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

	// Try as expression first, then as statements
	if strings.Contains(code, "\n") || strings.Contains(code, "return") {
		// Multi-line or contains return - execute as program
		modGlobals, execErr := starlark.ExecFileOptions(fileOptions, thread, "<eval>", code, predeclared)
		if execErr != nil {
			return &Result{Error: fmt.Sprintf("Execution error: %v", execErr)}, nil
		}
		// Look for a 'result' variable first, then fallback to all globals
		if resultVal, ok := modGlobals["result"]; ok {
			result = resultVal
		} else {
			// No explicit result variable - return filtered globals as a dict
			filteredGlobals := filterUserGlobals(modGlobals, predeclared)
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
		result, err = starlark.EvalOptions(fileOptions, thread, "<eval>", code, predeclared)
		if err != nil {
			return &Result{Error: fmt.Sprintf("Evaluation error: %v", err)}, nil
		}
	}

	// Convert result back to Go value
	goResult, err := StarlarkToGoValue(result)
	if err != nil {
		return &Result{Error: fmt.Sprintf("Result conversion error: %v", err)}, nil
	}

	return &Result{Result: goResult}, nil
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
