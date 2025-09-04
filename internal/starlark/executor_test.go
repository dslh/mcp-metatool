package starlark

import (
	"strings"
	"testing"
)

func TestExecute_SimpleExpressions(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		params   map[string]interface{}
		want     interface{}
		wantErr  bool
	}{
		{"arithmetic", "2 + 3", nil, int64(5), false},
		{"string concat", `"hello" + " world"`, nil, "hello world", false},
		{"boolean", "True", nil, true, false},
		{"list access", "[1, 2, 3][1]", nil, int64(2), false},
		{"dict access", `{"key": "value"}["key"]`, nil, "value", false},
		{"with params", `"Hello, " + params["name"]`, map[string]interface{}{"name": "World"}, "Hello, World", false},
		{"params arithmetic", `params["x"] * params["y"]`, map[string]interface{}{"x": 6, "y": 7}, int64(42), false},
		{"syntax error", "2 +", nil, nil, true},
		{"undefined variable", "undefined_var", nil, nil, true},
		{"division by zero", "1/0", nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(tt.code, tt.params)
			if err != nil {
				t.Errorf("Execute() framework error = %v", err)
				return
			}
			if tt.wantErr {
				if result == nil || result.Error == "" {
					t.Errorf("Execute() expected error in result, got none")
				}
				return
			}
			if result.Error != "" {
				t.Errorf("Execute() unexpected error in result: %s", result.Error)
				return
			}
			if result.Result != tt.want {
				t.Errorf("Execute() result = %v, want %v", result.Result, tt.want)
			}
		})
	}
}

func TestExecute_Programs_WithExplicitResult(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		params map[string]interface{}
		want   interface{}
	}{
		{
			"simple assignment",
			`x = 5
result = x * 2`,
			nil,
			int64(10),
		},
		{
			"string manipulation",
			`name = "Alice"
age = 30
result = "My name is " + name + " and I am " + str(age) + " years old"`,
			nil,
			"My name is Alice and I am 30 years old",
		},
		{
			"list processing",
			`data = [1, 2, 3, 4, 5]
processed = [x * 2 for x in data]
result = {"original": data, "processed": processed}`,
			nil,
			map[string]interface{}{
				"original":  []interface{}{int64(1), int64(2), int64(3), int64(4), int64(5)},
				"processed": []interface{}{int64(2), int64(4), int64(6), int64(8), int64(10)},
			},
		},
		{
			"using params",
			`multiplier = params["mult"]
numbers = params["nums"]
result = [n * multiplier for n in numbers]`,
			map[string]interface{}{
				"mult": 3,
				"nums": []interface{}{1, 2, 3},
			},
			[]interface{}{int64(3), int64(6), int64(9)},
		},
		{
			"simple calculation",
			`x = 5
y = 4
result = x * y * 3`,
			nil,
			int64(60),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(tt.code, tt.params)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if result.Error != "" {
				t.Errorf("Execute() error in result: %s", result.Error)
				return
			}
			if !equalValues(result.Result, tt.want) {
				t.Errorf("Execute() result = %v, want %v", result.Result, tt.want)
			}
		})
	}
}

func TestExecute_Programs_WithoutExplicitResult(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		params   map[string]interface{}
		wantVars []string // variables we expect in the result
	}{
		{
			"simple variables",
			`name = "Alice"
age = 30
city = "New York"`,
			nil,
			[]string{"name", "age", "city"},
		},
		{
			"with computations",
			`x = 10
y = 20
sum = x + y
product = x * y`,
			nil,
			[]string{"x", "y", "sum", "product"},
		},
		{
			"using params",
			`base = params["base"]
exponent = params["exp"]
power = base * base * base * base * base * base * base * base`,
			map[string]interface{}{"base": 2, "exp": 8},
			[]string{"base", "exponent", "power"},
		},
		{
			"list and dict variables",
			`numbers = [1, 2, 3, 4, 5]
squares = [n * n for n in numbers]
lookup = {"first": numbers[0], "last": numbers[-1]}`,
			nil,
			[]string{"numbers", "squares", "lookup"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(tt.code, tt.params)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if result.Error != "" {
				t.Errorf("Execute() error in result: %s", result.Error)
				return
			}

			// Result should be a map of variables
			resultMap, ok := result.Result.(map[string]interface{})
			if !ok {
				t.Errorf("Execute() result type = %T, want map[string]interface{}", result.Result)
				return
			}

			// Check that expected variables are present
			for _, varName := range tt.wantVars {
				if _, exists := resultMap[varName]; !exists {
					t.Errorf("Execute() missing expected variable: %s", varName)
				}
			}

			// Check that params are not included in result
			if tt.params != nil {
				if _, exists := resultMap["params"]; exists {
					t.Errorf("Execute() should not include 'params' in result variables")
				}
			}

			// Check that builtins are not included
			builtinNames := []string{"True", "False", "None", "print", "len", "str", "int"}
			for _, builtin := range builtinNames {
				if _, exists := resultMap[builtin]; exists {
					t.Errorf("Execute() should not include builtin '%s' in result", builtin)
				}
			}
		})
	}
}

func TestExecute_Programs_NoUserVariables(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		params map[string]interface{}
	}{
		{"only comments", "# This is a comment\n# Another comment", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(tt.code, tt.params)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if result.Error != "" {
				t.Errorf("Execute() error in result: %s", result.Error)
				return
			}
			// Should return None when no user variables
			if result.Result != nil {
				t.Errorf("Execute() result = %v, want nil (None)", result.Result)
			}
		})
	}
}

func TestExecute_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		params    map[string]interface{}
		wantError string // substring that should appear in error
	}{
		{"syntax error", "def broken(", nil, "error"},
		{"runtime error", "x = 1 / 0", nil, "error"},
		{"undefined variable", "result = undefined_var", nil, "error"},
		{"type error", `result = "string" + 42`, nil, "error"},
		{"invalid params", "result = params", map[string]interface{}{"invalid": make(chan int)}, "conversion error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(tt.code, tt.params)
			if err != nil {
				// Execution framework error
				if !strings.Contains(strings.ToLower(err.Error()), "error") {
					t.Errorf("Execute() error = %v, want error containing 'error'", err)
				}
				return
			}
			// Starlark execution error
			if result.Error == "" {
				t.Errorf("Execute() expected error in result, got none")
				return
			}
			if !strings.Contains(strings.ToLower(result.Error), strings.ToLower(tt.wantError)) {
				t.Errorf("Execute() error = %q, want error containing %q", result.Error, tt.wantError)
			}
		})
	}
}

func TestExecute_DetectExpressionVsProgram(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		isProgram   bool // true if should be executed as program
		expectsVars bool // true if we expect variables in result (program without explicit result)
	}{
		{"simple expression", "2 + 3", false, false},
		{"multiline expression", "2 + \\\n3", false, false}, // Line continuation
		{"program with newlines", "x = 1\ny = 2", true, true},
		{"program with result", "x = 1\nresult = x * 2", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(tt.code, nil)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if result.Error != "" {
				t.Errorf("Execute() error in result: %s", result.Error)
				return
			}

			if tt.expectsVars {
				// Should return a map of variables
				if _, ok := result.Result.(map[string]interface{}); !ok {
					t.Errorf("Execute() expected variables map, got %T", result.Result)
				}
			} else {
				// Should return a simple value or nil
				if resultMap, ok := result.Result.(map[string]interface{}); ok && len(resultMap) > 0 {
					t.Errorf("Execute() expected simple result, got variables map: %v", resultMap)
				}
			}
		})
	}
}

func TestExecute_ComplexPrograms(t *testing.T) {
	code := `
# Data processing example
data = [
    {"name": "Alice", "age": 30, "city": "NYC"},
    {"name": "Bob", "age": 25, "city": "LA"},
    {"name": "Charlie", "age": 35, "city": "NYC"}
]

# Filter NYC residents
nyc_residents = [person for person in data if person["city"] == "NYC"]

# Calculate average age (simplified)
avg_age = (nyc_residents[0]["age"] + nyc_residents[1]["age"]) / 2

# Build result
result = {
    "nyc_count": len(nyc_residents),
    "avg_age": avg_age,
    "names": [person["name"] for person in nyc_residents]
}
`

	result, err := Execute(code, nil)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
		return
	}
	if result.Error != "" {
		t.Errorf("Execute() error in result: %s", result.Error)
		return
	}

	expected := map[string]interface{}{
		"nyc_count": int64(2),
		"avg_age":   32.5,
		"names":     []interface{}{"Alice", "Charlie"},
	}

	if !equalValues(result.Result, expected) {
		t.Errorf("Execute() result = %v, want %v", result.Result, expected)
	}
}