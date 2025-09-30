package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	
	"github.com/dslh/mcp-metatool/internal/starlark"
)

func TestHandleEvalStarlark(t *testing.T) {
	tests := []struct {
		name       string
		args       EvalStarlarkArgs
		wantResult interface{}
		wantError  bool
	}{
		{
			"simple expression",
			EvalStarlarkArgs{Code: "2 + 3"},
			int64(5),
			false,
		},
		{
			"string expression",
			EvalStarlarkArgs{Code: `"hello" + " world"`},
			"hello world",
			false,
		},
		{
			"with parameters",
			EvalStarlarkArgs{
				Code:   `"Hello, " + params["name"]`,
				Params: map[string]interface{}{"name": "Alice"},
			},
			"Hello, Alice",
			false,
		},
		{
			"complex data structure",
			EvalStarlarkArgs{
				Code: `data = [1, 2, 3]
processed = [x * 2 for x in data]
result = {"original": data, "processed": processed}`,
			},
			map[string]interface{}{
				"original":  []interface{}{int64(1), int64(2), int64(3)},
				"processed": []interface{}{int64(2), int64(4), int64(6)},
			},
			false,
		},
		{
			"syntax error",
			EvalStarlarkArgs{Code: "2 +"},
			nil,
			true,
		},
		{
			"runtime error",
			EvalStarlarkArgs{Code: "undefined_var"},
			nil,
			true,
		},
		{
			"empty code",
			EvalStarlarkArgs{Code: ""},
			nil,
			true,
		},
		{
			"parameter conversion error",
			EvalStarlarkArgs{
				Code:   "result = params",
				Params: map[string]interface{}{"bad": make(chan int)},
			},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{} // Empty request is fine for testing

			result, returnValue, err := handleEvalStarlark(ctx, req, tt.args, nil)

			// Check for framework errors
			if err != nil {
				t.Errorf("handleEvalStarlark() framework error = %v", err)
				return
			}

			// Check result structure
			if result == nil {
				t.Errorf("handleEvalStarlark() result is nil")
				return
			}

			if len(result.Content) == 0 {
				t.Errorf("handleEvalStarlark() result has no content")
				return
			}

			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Errorf("handleEvalStarlark() result content is not TextContent")
				return
			}

			if tt.wantError {
				// Should contain error message
				if !containsAny(textContent.Text, []string{"error", "Error", "failed", "Failed"}) {
					t.Errorf("handleEvalStarlark() expected error message, got: %s", textContent.Text)
				}
				// Return value should be nil for errors
				if returnValue != nil {
					t.Errorf("handleEvalStarlark() expected nil return value for error, got: %v", returnValue)
				}
				return
			}

			// Should contain "Result:" for successful execution
			if !contains(textContent.Text, "Result:") {
				t.Errorf("handleEvalStarlark() expected success message, got: %s", textContent.Text)
				return
			}

			// Check return value matches expected result (returnValue is the Result struct)
			if resultStruct, ok := returnValue.(*starlark.Result); ok {
				if !equalValues(resultStruct.Result, tt.wantResult) {
					t.Errorf("handleEvalStarlark() returnValue.Result = %v, want %v", resultStruct.Result, tt.wantResult)
				}
			} else {
				t.Errorf("handleEvalStarlark() returnValue type = %T, want *starlark.Result", returnValue)
			}
		})
	}
}

func TestRegisterEvalStarlark(t *testing.T) {
	// Create a mock server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "0.1.0",
	}, nil)

	// Register the tool
	RegisterEvalStarlark(server, nil)

	// Verify the tool was registered
	// Note: The MCP SDK doesn't provide direct access to registered tools,
	// so we can only verify the registration doesn't panic
	t.Log("RegisterEvalStarlark completed without panic")
}

func TestEvalStarlarkArgs_Validation(t *testing.T) {
	tests := []struct {
		name string
		args EvalStarlarkArgs
		want string // expected validation error substring
	}{
		{
			"valid basic args",
			EvalStarlarkArgs{Code: "2 + 3"},
			"", // no error expected
		},
		{
			"valid with params",
			EvalStarlarkArgs{
				Code:   "result = params['x']",
				Params: map[string]interface{}{"x": 42},
			},
			"", // no error expected
		},
		{
			"empty code should be handled gracefully",
			EvalStarlarkArgs{Code: ""},
			"", // framework should handle this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, _, err := handleEvalStarlark(ctx, req, tt.args, nil)

			if err != nil {
				if tt.want == "" {
					t.Errorf("handleEvalStarlark() unexpected error = %v", err)
				} else if !contains(err.Error(), tt.want) {
					t.Errorf("handleEvalStarlark() error = %v, want error containing %s", err, tt.want)
				}
				return
			}

			if tt.want != "" {
				// Expected an error but didn't get one - check if it's in result
				textContent := result.Content[0].(*mcp.TextContent)
				if !contains(textContent.Text, tt.want) {
					t.Errorf("handleEvalStarlark() expected error containing %s, got: %s", tt.want, textContent.Text)
				}
			}
		})
	}
}

func TestEvalStarlarkIntegration(t *testing.T) {
	// Test more complex scenarios that integrate multiple features
	tests := []struct {
		name string
		args EvalStarlarkArgs
		want interface{}
	}{
		{
			"fibonacci calculation",
			EvalStarlarkArgs{
				Code: `# Calculate fibonacci sequence with for loop
fib = [0, 1]
for i in range(8):
    fib.append(fib[-1] + fib[-2])
result = fib`,
			},
			[]interface{}{int64(0), int64(1), int64(1), int64(2), int64(3), int64(5), int64(8), int64(13), int64(21), int64(34)},
		},
		{
			"data transformation with params",
			EvalStarlarkArgs{
				Code: `users = params["users"]
adults = [user for user in users if user["age"] >= 18]
result = {
    "total": len(users),
    "adults": len(adults),
    "adult_names": [user["name"] for user in adults]
}`,
				Params: map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{"name": "Alice", "age": 25},
						map[string]interface{}{"name": "Bob", "age": 16},
						map[string]interface{}{"name": "Charlie", "age": 30},
					},
				},
			},
			map[string]interface{}{
				"total":      int64(3),
				"adults":     int64(2),
				"adult_names": []interface{}{"Alice", "Charlie"},
			},
		},
		{
			"string processing",
			EvalStarlarkArgs{
				Code: `text = params["text"]
words = text.split()
result = {
    "word_count": len(words),
    "char_count": len(text),
    "words": words,
    "reversed": text[::-1]
}`,
				Params: map[string]interface{}{
					"text": "Hello World",
				},
			},
			map[string]interface{}{
				"word_count": int64(2),
				"char_count": int64(11),
				"words":      []interface{}{"Hello", "World"},
				"reversed":   "dlroW olleH",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, returnValue, err := handleEvalStarlark(ctx, req, tt.args, nil)

			if err != nil {
				t.Errorf("handleEvalStarlark() error = %v", err)
				return
			}

			textContent := result.Content[0].(*mcp.TextContent)
			if !contains(textContent.Text, "Result:") {
				t.Errorf("handleEvalStarlark() expected success, got: %s", textContent.Text)
				return
			}

			if resultStruct, ok := returnValue.(*starlark.Result); ok {
				if !equalValues(resultStruct.Result, tt.want) {
					t.Errorf("handleEvalStarlark() returnValue.Result = %v, want %v", resultStruct.Result, tt.want)
				}
			} else {
				t.Errorf("handleEvalStarlark() returnValue type = %T, want *starlark.Result", returnValue)
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// equalValues compares two values, handling type differences between Go and Starlark
func equalValues(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle int/int64 conversion
	if aInt, ok := a.(int); ok {
		if bInt64, ok := b.(int64); ok {
			return int64(aInt) == bInt64
		}
	}
	if aInt64, ok := a.(int64); ok {
		if bInt, ok := b.(int); ok {
			return aInt64 == int64(bInt)
		}
	}

	// Handle slices
	if aSlice, ok := a.([]interface{}); ok {
		if bSlice, ok := b.([]interface{}); ok {
			if len(aSlice) != len(bSlice) {
				return false
			}
			for i := range aSlice {
				if !equalValues(aSlice[i], bSlice[i]) {
					return false
				}
			}
			return true
		}
		return false
	}

	// Handle maps
	if aMap, ok := a.(map[string]interface{}); ok {
		if bMap, ok := b.(map[string]interface{}); ok {
			if len(aMap) != len(bMap) {
				return false
			}
			for k, v := range aMap {
				if !equalValues(v, bMap[k]) {
					return false
				}
			}
			return true
		}
		return false
	}

	return a == b
}