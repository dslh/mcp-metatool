package starlark

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

func TestExecute_FlowControl(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		params map[string]interface{}
		want   interface{}
	}{
		{
			"simple if/else",
			`x = 10
if x > 5:
    result = "greater"
else:
    result = "lesser"`,
			nil,
			"greater",
		},
		{
			"nested if statements",
			`score = 85
if score >= 90:
    grade = "A"
elif score >= 80:
    grade = "B"
elif score >= 70:
    grade = "C"
else:
    grade = "F"
result = grade`,
			nil,
			"B",
		},
		{
			"while loop",
			`i = 0
total = 0
while i < 5:
    total += i
    i += 1
result = total`,
			nil,
			int64(10),
		},
		{
			"while with break",
			`i = 0
numbers = []
while True:
    if i >= 3:
        break
    numbers.append(i)
    i += 1
result = numbers`,
			nil,
			[]interface{}{int64(0), int64(1), int64(2)},
		},
		{
			"for loop with range",
			`squares = []
for i in range(4):
    squares.append(i * i)
result = squares`,
			nil,
			[]interface{}{int64(0), int64(1), int64(4), int64(9)},
		},
		{
			"for loop with list",
			`words = ["hello", "world", "test"]
lengths = []
for word in words:
    lengths.append(len(word))
result = lengths`,
			nil,
			[]interface{}{int64(5), int64(5), int64(4)},
		},
		{
			"for loop with continue",
			`evens = []
for i in range(10):
    if i % 2 == 1:
        continue
    evens.append(i)
result = evens`,
			nil,
			[]interface{}{int64(0), int64(2), int64(4), int64(6), int64(8)},
		},
		{
			"for loop with break",
			`result = 0
for i in range(100):
    result += i
    if i >= 5:
        break`,
			nil,
			int64(15),
		},
		{
			"nested loops",
			`matrix = []
for i in range(3):
    row = []
    for j in range(3):
        row.append(i * 3 + j)
    matrix.append(row)
result = matrix`,
			nil,
			[]interface{}{
				[]interface{}{int64(0), int64(1), int64(2)},
				[]interface{}{int64(3), int64(4), int64(5)},
				[]interface{}{int64(6), int64(7), int64(8)},
			},
		},
		{
			"pass statement",
			`x = 5
if x > 0:
    pass
else:
    x = -1
result = x`,
			nil,
			int64(5),
		},
		{
			"conditional with params",
			`threshold = params["threshold"]
value = params["value"]
if value > threshold:
    result = "high"
else:
    result = "low"`,
			map[string]interface{}{"threshold": 50, "value": 75},
			"high",
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

func TestExecute_BuiltinFunctions(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		params map[string]interface{}
		want   interface{}
	}{
		// Math functions
		{
			"abs function",
			`numbers = [-5, 3.14, -2.7]
result = [abs(x) for x in numbers]`,
			nil,
			[]interface{}{int64(5), 3.14, 2.7},
		},
		{
			"max and min",
			`numbers = [3, 1, 4, 1, 5, 9]
result = {"max": max(numbers), "min": min(numbers)}`,
			nil,
			map[string]interface{}{"max": int64(9), "min": int64(1)},
		},
		
		// Type functions
		{
			"type function",
			`values = [42, "hello", [1, 2, 3], {"a": 1}]
result = [type(x) for x in values]`,
			nil,
			[]interface{}{"int", "string", "list", "dict"},
		},
		{
			"hasattr and getattr",
			`d = {"name": "test", "value": 42}
result = {"has_name": hasattr(d, "get"), "has_nonexistent": hasattr(d, "nonexistent")}`,
			nil,
			map[string]interface{}{"has_name": true, "has_nonexistent": false},
		},
		
		// Sequence functions
		{
			"enumerate",
			`items = ["a", "b", "c"]
result = [(i, v) for i, v in enumerate(items)]`,
			nil,
			[]interface{}{
				[]interface{}{int64(0), "a"},
				[]interface{}{int64(1), "b"},
				[]interface{}{int64(2), "c"},
			},
		},
		{
			"reversed",
			`list(reversed([1, 2, 3, 4, 5]))`,
			nil,
			[]interface{}{int64(5), int64(4), int64(3), int64(2), int64(1)},
		},
		{
			"sorted",
			`sorted([3, 1, 4, 1, 5, 9, 2, 6])`,
			nil,
			[]interface{}{int64(1), int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(9)},
		},
		{
			"zip",
			`names = ["Alice", "Bob", "Charlie"]
ages = [30, 25, 35]
result = [(name, age) for name, age in zip(names, ages)]`,
			nil,
			[]interface{}{
				[]interface{}{"Alice", int64(30)},
				[]interface{}{"Bob", int64(25)},
				[]interface{}{"Charlie", int64(35)},
			},
		},
		
		// Conversion functions
		{
			"chr and ord",
			`{"chr_65": chr(65), "ord_A": ord("A"), "chr_97": chr(97)}`,
			nil,
			map[string]interface{}{"chr_65": "A", "ord_A": int64(65), "chr_97": "a"},
		},
		{
			"bool conversion",
			`[bool(1), bool(0), bool(""), bool("hello"), bool([]), bool([1])]`,
			nil,
			[]interface{}{true, false, false, true, false, true},
		},
		{
			"float conversion",
			`[float(42), float("3.14"), float("-2.5")]`,
			nil,
			[]interface{}{42.0, 3.14, -2.5},
		},
		{
			"int conversion",
			`[int(3.14), int("42"), int("-17")]`,
			nil,
			[]interface{}{int64(3), int64(42), int64(-17)},
		},
		{
			"str conversion",
			`[str(42), str(3.14), str(True), str([1, 2, 3])]`,
			nil,
			[]interface{}{"42", "3.14", "True", "[1, 2, 3]"},
		},
		
		// Collection functions
		{
			"any and all",
			`result = {
    "any_mixed": any([False, 0, "hello"]),
    "any_empty": any([]),
    "all_true": all([True, 1, "hello"]),
    "all_mixed": all([True, 0, "hello"])
}`,
			nil,
			map[string]interface{}{
				"any_mixed": true,
				"any_empty": false,
				"all_true":  true,
				"all_mixed": false,
			},
		},
		{
			"set operations",
			`result = {
    "set_from_list": sorted(list(set([1, 2, 2, 3, 3, 3]))),
    "set_length": len(set([1, 1, 2, 2, 3, 3]))
}`,
			nil,
			map[string]interface{}{
				"set_from_list": []interface{}{int64(1), int64(2), int64(3)},
				"set_length":    int64(3),
			},
		},
		
		// Utility functions
		{
			"repr function",
			`[repr("hello"), repr([1, 2, 3]), repr({"key": "value"})]`,
			nil,
			[]interface{}{"\"hello\"", "[1, 2, 3]", "{\"key\": \"value\"}"},
		},
		{
			"range function",
			`result = {
    "range_5": list(range(5)),
    "range_2_7": list(range(2, 7)),
    "range_step": list(range(0, 10, 2))
}`,
			nil,
			map[string]interface{}{
				"range_5":    []interface{}{int64(0), int64(1), int64(2), int64(3), int64(4)},
				"range_2_7":  []interface{}{int64(2), int64(3), int64(4), int64(5), int64(6)},
				"range_step": []interface{}{int64(0), int64(2), int64(4), int64(6), int64(8)},
			},
		},
		
		// Complex combinations
		{
			"combined builtins",
			`data = [1, -2, 3, -4, 5]
positive = [x for x in data if x > 0]
result = {
    "original": data,
    "positive": positive,
    "abs_values": [abs(x) for x in data],
    "max_positive": max(positive),
    "sorted_desc": sorted(data, reverse=True)
}`,
			nil,
			map[string]interface{}{
				"original":    []interface{}{int64(1), int64(-2), int64(3), int64(-4), int64(5)},
				"positive":    []interface{}{int64(1), int64(3), int64(5)},
				"abs_values":  []interface{}{int64(1), int64(2), int64(3), int64(4), int64(5)},
				"max_positive": int64(5),
				"sorted_desc": []interface{}{int64(5), int64(3), int64(1), int64(-2), int64(-4)},
			},
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

func TestExecute_AdvancedFeatures(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		params map[string]interface{}
		want   interface{}
	}{
		// Set operations and comprehensions
		{
			"set literals and operations",
			`s1 = set([1, 2, 3, 4])
s2 = set([3, 4, 5, 6])
result = {
    "s1_list": sorted(list(s1)),
    "s2_list": sorted(list(s2)),
    "union": sorted(list(s1 | s2)),
    "intersection": sorted(list(s1 & s2)),
    "difference": sorted(list(s1 - s2))
}`,
			nil,
			map[string]interface{}{
				"s1_list":      []interface{}{int64(1), int64(2), int64(3), int64(4)},
				"s2_list":      []interface{}{int64(3), int64(4), int64(5), int64(6)},
				"union":        []interface{}{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)},
				"intersection": []interface{}{int64(3), int64(4)},
				"difference":   []interface{}{int64(1), int64(2)},
			},
		},
		{
			"set comprehension",
			`numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
even_squares = set([x * x for x in numbers if x % 2 == 0])
result = sorted(list(even_squares))`,
			nil,
			[]interface{}{int64(4), int64(16), int64(36), int64(64), int64(100)},
		},
		
		// Complex comprehensions
		{
			"nested list comprehension",
			`matrix = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
flattened = [item for row in matrix for item in row]
result = flattened`,
			nil,
			[]interface{}{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8), int64(9)},
		},
		{
			"dict comprehension",
			`words = ["hello", "world", "python", "starlark"]
word_lengths = {word: len(word) for word in words if len(word) > 4}
result = word_lengths`,
			nil,
			map[string]interface{}{
				"hello":    int64(5),
				"world":    int64(5),
				"python":   int64(6),
				"starlark": int64(8),
			},
		},
		{
			"conditional comprehension",
			`data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
result = [x * 2 if x % 2 == 0 else x for x in data]`,
			nil,
			[]interface{}{int64(1), int64(4), int64(3), int64(8), int64(5), int64(12), int64(7), int64(16), int64(9), int64(20)},
		},
		
		// Multiple assignment and unpacking
		{
			"tuple unpacking",
			`coords = (3, 4)
x, y = coords
distance_squared = x*x + y*y
result = {"x": x, "y": y, "distance_squared": distance_squared}`,
			nil,
			map[string]interface{}{"x": int64(3), "y": int64(4), "distance_squared": int64(25)},
		},
		{
			"multiple assignment",
			`a, b, c = 1, 2, 3
result = [a, b, c]`,
			nil,
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"variable swapping",
			`a = 10
b = 20
a, b = b, a
result = {"a": a, "b": b}`,
			nil,
			map[string]interface{}{"a": int64(20), "b": int64(10)},
		},
		{
			"list unpacking",
			`numbers = [1, 2, 3, 4, 5]
first, second = numbers[0], numbers[1]
rest = numbers[2:]
result = {"first": first, "second": second, "rest": rest}`,
			nil,
			map[string]interface{}{
				"first":  int64(1),
				"second": int64(2),
				"rest":   []interface{}{int64(3), int64(4), int64(5)},
			},
		},
		
		// Nested structures
		{
			"deeply nested data",
			`data = {
    "users": [
        {"name": "Alice", "scores": [85, 92, 78], "metadata": {"active": True}},
        {"name": "Bob", "scores": [90, 88, 95], "metadata": {"active": False}},
        {"name": "Charlie", "scores": [76, 84, 89], "metadata": {"active": True}}
    ]
}
active_users = [user for user in data["users"] if user["metadata"]["active"]]
avg_scores = {}
for user in active_users:
    total = 0
    for score in user["scores"]:
        total += score
    avg_scores[user["name"]] = total / len(user["scores"])
result = avg_scores`,
			nil,
			map[string]interface{}{
				"Alice":   85.0,
				"Charlie": 83.0,
			},
		},
		
		// Advanced string operations
		{
			"string methods and formatting",
			`template = "Hello, {name}! You have {count} messages."
users = [
    {"name": "Alice", "count": 5},
    {"name": "Bob", "count": 0},
    {"name": "Charlie", "count": 3}
]
result = [template.format(name=user["name"], count=user["count"]) for user in users if user["count"] > 0]`,
			nil,
			[]interface{}{
				"Hello, Alice! You have 5 messages.",
				"Hello, Charlie! You have 3 messages.",
			},
		},
		
		// Complex filtering and transformation
		{
			"data pipeline",
			`raw_data = [
    {"id": 1, "value": 10, "category": "A", "valid": True},
    {"id": 2, "value": 25, "category": "B", "valid": False},
    {"id": 3, "value": 15, "category": "A", "valid": True},
    {"id": 4, "value": 30, "category": "B", "valid": True},
    {"id": 5, "value": 8, "category": "A", "valid": False}
]

# Multi-step pipeline
valid_data = [item for item in raw_data if item["valid"]]
by_category = {}
for item in valid_data:
    cat = item["category"]
    if cat not in by_category:
        by_category[cat] = []
    by_category[cat].append(item["value"])

category_sums = {}
for cat, values in by_category.items():
    total = 0
    for val in values:
        total += val
    category_sums[cat] = total

result = {
    "category_sums": category_sums,
    "category_counts": {cat: len(values) for cat, values in by_category.items()},
    "max_value": max([item["value"] for item in valid_data])
}`,
			nil,
			map[string]interface{}{
				"category_sums":   map[string]interface{}{"A": int64(25), "B": int64(30)},
				"category_counts": map[string]interface{}{"A": int64(2), "B": int64(1)},
				"max_value":       int64(30),
			},
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

func TestExecute_ComplexIntegrationScenarios(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		params map[string]interface{}
		want   interface{}
	}{
		{
			"log analysis pipeline",
			`
# Simulate processing log entries
logs = [
    {"timestamp": "2023-01-01T10:00:00", "level": "INFO", "message": "Server started", "service": "web"},
    {"timestamp": "2023-01-01T10:01:00", "level": "ERROR", "message": "Database connection failed", "service": "api"},
    {"timestamp": "2023-01-01T10:02:00", "level": "WARN", "message": "High memory usage", "service": "web"},
    {"timestamp": "2023-01-01T10:03:00", "level": "INFO", "message": "User login", "service": "auth"},
    {"timestamp": "2023-01-01T10:04:00", "level": "ERROR", "message": "Payment failed", "service": "payment"},
    {"timestamp": "2023-01-01T10:05:00", "level": "DEBUG", "message": "Query executed", "service": "api"}
]

# Analysis pipeline
error_logs = [log for log in logs if log["level"] == "ERROR"]
warning_logs = [log for log in logs if log["level"] == "WARN"]

service_stats = {}
for log in logs:
    service = log["service"]
    level = log["level"]
    if service not in service_stats:
        service_stats[service] = {"total": 0, "errors": 0, "warnings": 0}
    
    service_stats[service]["total"] += 1
    if level == "ERROR":
        service_stats[service]["errors"] += 1
    elif level == "WARN":
        service_stats[service]["warnings"] += 1

# Generate report
critical_services = [service for service, stats in service_stats.items() 
                   if stats["errors"] > 0 or stats["warnings"] > 0]

result = {
    "total_logs": len(logs),
    "error_count": len(error_logs),
    "critical_services": sorted(critical_services),
    "service_health": {service: stats["errors"] == 0 for service, stats in service_stats.items()},
    "error_messages": [log["message"] for log in error_logs]
}
`,
			nil,
			map[string]interface{}{
				"total_logs":       int64(6),
				"error_count":      int64(2),
				"critical_services": []interface{}{"api", "payment", "web"},
				"service_health":   map[string]interface{}{"api": false, "auth": true, "payment": false, "web": true},
				"error_messages":   []interface{}{"Database connection failed", "Payment failed"},
			},
		},
		{
			"machine learning data preprocessing",
			`
# Simulate ML data preprocessing
raw_features = [
    {"age": 25, "income": 50000, "education": "bachelor", "score": 0.8},
    {"age": 35, "income": 75000, "education": "master", "score": 0.9},
    {"age": 45, "income": 60000, "education": "bachelor", "score": 0.7},
    {"age": 30, "income": 80000, "education": "phd", "score": 0.95},
    {"age": 28, "income": 45000, "education": "bachelor", "score": 0.75}
]

# Normalization and encoding
education_map = {"bachelor": 1, "master": 2, "phd": 3}
ages = [f["age"] for f in raw_features]
incomes = [f["income"] for f in raw_features]

min_age, max_age = min(ages), max(ages)
min_income, max_income = min(incomes), max(incomes)

processed_features = []
for feature in raw_features:
    normalized_age = (feature["age"] - min_age) / (max_age - min_age)
    normalized_income = (feature["income"] - min_income) / (max_income - min_income)
    education_encoded = education_map[feature["education"]]
    
    processed_features.append({
        "normalized_age": int(normalized_age * 1000) / 1000.0,
        "normalized_income": int(normalized_income * 1000) / 1000.0,
        "education_level": education_encoded,
        "score": feature["score"]
    })

# Feature statistics
avg_score_by_education = {}
for level_name, level_value in education_map.items():
    scores = [f["score"] for f in raw_features if education_map[f["education"]] == level_value]
    if scores:
        total = 0
        for score in scores:
            total += score
        avg_score_by_education[level_name] = total / len(scores)
    else:
        avg_score_by_education[level_name] = 0

result = {
    "processed_count": len(processed_features),
    "feature_ranges": {
        "age_range": [min_age, max_age],
        "income_range": [min_income, max_income]
    },
    "education_performance": avg_score_by_education,
    "high_performers": len([f for f in raw_features if f["score"] >= 0.9])
}
`,
			nil,
			map[string]interface{}{
				"processed_count": int64(5),
				"feature_ranges": map[string]interface{}{
					"age_range":    []interface{}{int64(25), int64(45)},
					"income_range": []interface{}{int64(45000), int64(80000)},
				},
				"education_performance": map[string]interface{}{
					"bachelor": 0.75,
					"master":   0.9,
					"phd":      0.95,
				},
				"high_performers": int64(2),
			},
		},
		{
			"e-commerce analytics",
			`
# Simulate e-commerce order analysis
orders = [
    {"id": 1, "user_id": 101, "items": [{"product": "laptop", "price": 1000, "qty": 1}], "status": "completed"},
    {"id": 2, "user_id": 102, "items": [{"product": "mouse", "price": 25, "qty": 2}, {"product": "keyboard", "price": 75, "qty": 1}], "status": "completed"},
    {"id": 3, "user_id": 101, "items": [{"product": "monitor", "price": 300, "qty": 1}], "status": "cancelled"},
    {"id": 4, "user_id": 103, "items": [{"product": "laptop", "price": 1000, "qty": 1}, {"product": "mouse", "price": 25, "qty": 1}], "status": "completed"},
    {"id": 5, "user_id": 102, "items": [{"product": "keyboard", "price": 75, "qty": 2}], "status": "pending"}
]

# Revenue analysis
completed_orders = [order for order in orders if order["status"] == "completed"]
total_revenue = 0
product_sales = {}
user_spending = {}

for order in completed_orders:
    user_id = order["user_id"]
    order_total = 0
    
    for item in order["items"]:
        item_total = item["price"] * item["qty"]
        order_total += item_total
        product = item["product"]
        
        if product not in product_sales:
            product_sales[product] = {"revenue": 0, "quantity": 0}
        product_sales[product]["revenue"] += item_total
        product_sales[product]["quantity"] += item["qty"]
    
    total_revenue += order_total
    if user_id not in user_spending:
        user_spending[user_id] = 0
    user_spending[user_id] += order_total

# Top products and users
top_product = max(product_sales.items(), key=lambda x: x[1]["revenue"])
top_user = max(user_spending.items(), key=lambda x: x[1])

result = {
    "total_orders": len(orders),
    "completed_orders": len(completed_orders),
    "total_revenue": total_revenue,
    "average_order_value": total_revenue / len(completed_orders) if completed_orders else 0,
    "top_product": {"name": top_product[0], "revenue": top_product[1]["revenue"]},
    "top_user": {"id": top_user[0], "spending": top_user[1]},
    "product_count": len(product_sales)
}
`,
			nil,
			map[string]interface{}{
				"total_orders":       int64(5),
				"completed_orders":   int64(3),
				"total_revenue":      int64(2150),
				"average_order_value": 716.6666666666666,
				"top_product":        map[string]interface{}{"name": "laptop", "revenue": int64(2000)},
				"top_user":           map[string]interface{}{"id": int64(103), "spending": int64(1025)},
				"product_count":      int64(3),
			},
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

func TestExecuteWithProxy(t *testing.T) {
	// Test without proxy manager (should work like normal Execute)
	result, err := ExecuteWithProxy("2 + 3", nil, nil)
	if err != nil {
		t.Errorf("Execute without proxy failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}
	if result.Result != int64(5) {
		t.Errorf("Expected 5, got %v", result.Result)
	}

	// Test with proxy manager
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("echo", []*mcp.Tool{
		{Name: "test", Description: "Test tool"},
	})

	// Test that server namespace is available
	code := `
result = echo
result
`
	result, err = ExecuteWithProxy(code, nil, mockProxy)
	if err != nil {
		t.Errorf("Execute with proxy failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}

	// The result should be the server namespace object
	if result.Result == nil {
		t.Error("Expected result to contain server namespace")
	}
}

func TestExecuteWithProxyAndParams(t *testing.T) {
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("test", []*mcp.Tool{
		{Name: "greet", Description: "Greeting tool"},
	})

	params := map[string]interface{}{
		"name": "Alice",
		"count": 3,
	}

	code := `
# Test that both params and server namespace are available
name = params["name"]
result = {
    "greeting": name + " says hello",
    "server_available": test != None
}
result
`

	result, err := ExecuteWithProxy(code, params, mockProxy)
	if err != nil {
		t.Errorf("Execute with proxy and params failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}

	// Check the result structure
	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be a map, got %T", result.Result)
	}

	if resultMap["greeting"] != "Alice says hello" {
		t.Errorf("Expected greeting='Alice says hello', got %v", resultMap["greeting"])
	}

	if resultMap["server_available"] != true {
		t.Errorf("Expected server_available=true, got %v", resultMap["server_available"])
	}
}

func TestExecuteWithProxyToolCall(t *testing.T) {
	mockProxy := NewMockProxyManager()
	mockProxy.AddServer("echo", []*mcp.Tool{
		{Name: "echo", Description: "Echo tool"},
	})

	code := `
result = echo.echo({"message": "test"})
result
`

	result, err := ExecuteWithProxy(code, nil, mockProxy)
	if err != nil {
		t.Errorf("Execute with proxy tool call failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}

	// Verify that the tool was called
	if len(mockProxy.calls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(mockProxy.calls))
	}

	call := mockProxy.calls[0]
	if call.ServerName != "echo" || call.ToolName != "echo" {
		t.Errorf("Unexpected call: %+v", call)
	}
	if call.Arguments["message"] != "test" {
		t.Errorf("Expected message='test', got %v", call.Arguments["message"])
	}

	// Check that result contains tool response
	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be a map, got %T", result.Result)
	}

	if _, hasContent := resultMap["content"]; !hasContent {
		t.Error("Expected result to have 'content' key")
	}
	if _, hasStructured := resultMap["structured"]; !hasStructured {
		t.Error("Expected result to have 'structured' key")
	}
}

func TestExecuteBackwardCompatibility(t *testing.T) {
	// Test that the original Execute function still works
	result, err := Execute("10 * 4", nil)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}
	if result.Result != int64(40) {
		t.Errorf("Expected 40, got %v", result.Result)
	}

	// Test with parameters
	params := map[string]interface{}{
		"x": 5,
		"y": 7,
	}

	result, err = Execute("params['x'] + params['y']", params)
	if err != nil {
		t.Errorf("Execute with params failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}
	if result.Result != int64(12) {
		t.Errorf("Expected 12, got %v", result.Result)
	}
}

func TestExecute_TimeModule(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		params  map[string]interface{}
		wantErr bool
		check   func(t *testing.T, result interface{})
	}{
		{
			name:    "time.now returns a time",
			code:    "time.now()",
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				if result == nil {
					t.Error("time.now() should return a value")
				}
			},
		},
		{
			name:    "time.parse_time parses ISO 8601",
			code:    `time.parse_time("2025-01-15T10:30:00Z")`,
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				if result == nil {
					t.Error("time.parse_time() should return a time value")
				}
			},
		},
		{
			name:    "time.parse_time with format",
			code:    `time.parse_time("2025-01-15", "2006-01-02")`,
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				if result == nil {
					t.Error("time.parse_time() with format should return a time value")
				}
			},
		},
		{
			name:    "time.time creates time",
			code:    `time.time(year=2025, month=1, day=15, hour=10, minute=30, second=0)`,
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				if result == nil {
					t.Error("time.time() should return a time value")
				}
			},
		},
		{
			name:    "time constants",
			code:    `{"hour": time.hour, "minute": time.minute, "second": time.second}`,
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				if resultMap["hour"] == nil || resultMap["minute"] == nil || resultMap["second"] == nil {
					t.Error("Time constants should be defined")
				}
			},
		},
		{
			name:    "time.parse_duration",
			code:    `time.parse_duration("1h30m")`,
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				if result == nil {
					t.Error("time.parse_duration() should return a duration")
				}
			},
		},
		{
			name:    "time.is_valid_timezone",
			code:    `{"utc": time.is_valid_timezone("UTC"), "invalid": time.is_valid_timezone("NotATimezone")}`,
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				if resultMap["utc"] != true {
					t.Error("UTC should be a valid timezone")
				}
				if resultMap["invalid"] != false {
					t.Error("NotATimezone should not be valid")
				}
			},
		},
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
			if tt.check != nil {
				tt.check(t, result.Result)
			}
		})
	}
}

func TestExecute_MathModule(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		params  map[string]interface{}
		want    interface{}
		wantErr bool
		check   func(t *testing.T, result interface{})
	}{
		{
			name: "math.sqrt",
			code: "math.sqrt(16)",
			want: 4.0,
		},
		{
			name: "math.ceil",
			code: "math.ceil(3.2)",
			want: int64(4),
		},
		{
			name: "math.floor",
			code: "math.floor(3.8)",
			want: int64(3),
		},
		{
			name: "math.pow",
			code: "math.pow(2, 3)",
			want: 8.0,
		},
		{
			name: "math.round",
			code: "math.round(3.6)",
			want: 4.0,
		},
		{
			name: "math.fabs",
			code: "math.fabs(-5.5)",
			want: 5.5,
		},
		{
			name: "math constants",
			code: "math.pi > 3.14 and math.pi < 3.15",
			want: true,
		},
		{
			name: "math.e constant",
			code: "math.e > 2.71 and math.e < 2.72",
			want: true,
		},
		{
			name: "math.sin",
			code: "math.sin(0)",
			want: 0.0,
		},
		{
			name: "math.cos",
			code: "math.cos(0)",
			want: 1.0,
		},
		{
			name: "math.exp",
			code: "math.exp(0)",
			want: 1.0,
		},
		{
			name: "math.log natural log",
			code: "math.log(math.e)",
			want: 1.0,
		},
		{
			name: "math.log with base",
			code: "math.log(100, 10)",
			want: 2.0,
		},
		{
			name: "math.degrees",
			code: "math.degrees(math.pi)",
			want: 180.0,
		},
		{
			name: "math.radians",
			code: "math.radians(180)",
			want: nil, // Will check in the check function
			check: func(t *testing.T, result interface{}) {
				f, ok := result.(float64)
				if !ok {
					t.Errorf("Expected float64, got %T", result)
					return
				}
				// Check if it's approximately pi
				if f < 3.14 || f > 3.15 {
					t.Errorf("math.radians(180) = %v, expected approximately pi (3.14159)", f)
				}
			},
		},
		{
			name: "math operations in expressions",
			code: "math.sqrt(math.pow(3, 2) + math.pow(4, 2))",
			want: 5.0,
		},
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
			if tt.check != nil {
				tt.check(t, result.Result)
			} else if tt.want != nil && result.Result != tt.want {
				t.Errorf("Execute() result = %v (type %T), want %v (type %T)", result.Result, result.Result, tt.want, tt.want)
			}
		})
	}
}

func TestExecute_JsonModule(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		params  map[string]interface{}
		want    interface{}
		wantErr bool
		check   func(t *testing.T, result interface{})
	}{
		{
			name: "json.encode dict",
			code: `json.encode({"key": "value", "number": 42})`,
			check: func(t *testing.T, result interface{}) {
				jsonStr, ok := result.(string)
				if !ok {
					t.Errorf("Expected string, got %T", result)
					return
				}
				// JSON can order keys differently, so check if it contains the expected parts
				if !strings.Contains(jsonStr, `"key"`) || !strings.Contains(jsonStr, `"value"`) {
					t.Errorf("json.encode result doesn't contain expected content: %s", jsonStr)
				}
			},
		},
		{
			name: "json.encode list",
			code: `json.encode([1, 2, 3, "four"])`,
			want: `[1,2,3,"four"]`,
		},
		{
			name: "json.encode string",
			code: `json.encode("hello world")`,
			want: `"hello world"`,
		},
		{
			name: "json.encode number",
			code: `json.encode(42)`,
			want: "42",
		},
		{
			name: "json.encode bool",
			code: `json.encode(True)`,
			want: "true",
		},
		{
			name: "json.encode null",
			code: `json.encode(None)`,
			want: "null",
		},
		{
			name: "json.decode object",
			code: `json.decode('{"x": 42, "y": "test"}')`,
			check: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				if resultMap["x"] != int64(42) {
					t.Errorf("Expected x=42, got %v", resultMap["x"])
				}
				if resultMap["y"] != "test" {
					t.Errorf("Expected y=test, got %v", resultMap["y"])
				}
			},
		},
		{
			name: "json.decode array",
			code: `json.decode('[1, 2, 3]')`,
			want: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name: "json.decode string",
			code: `json.decode('"hello"')`,
			want: "hello",
		},
		{
			name: "json.decode number",
			code: `json.decode('123')`,
			want: int64(123),
		},
		{
			name: "json.decode float",
			code: `json.decode('3.14')`,
			want: 3.14,
		},
		{
			name: "json.decode bool",
			code: `json.decode('true')`,
			want: true,
		},
		{
			name: "json.decode null",
			code: `json.decode('null')`,
			want: nil,
		},
		{
			name: "json.indent",
			code: `json.indent('{"a":1,"b":2}')`,
			check: func(t *testing.T, result interface{}) {
				jsonStr, ok := result.(string)
				if !ok {
					t.Errorf("Expected string, got %T", result)
					return
				}
				// Check that it contains indentation (tabs or spaces)
				if !strings.Contains(jsonStr, "\t") && !strings.Contains(jsonStr, "  ") {
					t.Errorf("json.indent should add indentation: %s", jsonStr)
				}
			},
		},
		{
			name: "json encode-decode round trip",
			code: `data = {"name": "test", "values": [1, 2, 3], "nested": {"key": "value"}}
encoded = json.encode(data)
decoded = json.decode(encoded)
result = decoded`,
			check: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				if resultMap["name"] != "test" {
					t.Errorf("Round trip failed: name = %v", resultMap["name"])
				}
			},
		},
		{
			name: "json.decode with nested structures",
			code: `json.decode('{"users": [{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]}')`,
			check: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				users, ok := resultMap["users"].([]interface{})
				if !ok {
					t.Errorf("Expected users to be array")
					return
				}
				if len(users) != 2 {
					t.Errorf("Expected 2 users, got %d", len(users))
				}
			},
		},
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
			if tt.check != nil {
				tt.check(t, result.Result)
			} else if tt.want != nil {
				// For slices, need to do deep comparison
				if !deepEqual(result.Result, tt.want) {
					t.Errorf("Execute() result = %v (type %T), want %v (type %T)", result.Result, result.Result, tt.want, tt.want)
				}
			}
		})
	}
}

// Helper function for deep comparison
func deepEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle slices
	aSlice, aIsSlice := a.([]interface{})
	bSlice, bIsSlice := b.([]interface{})
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}
		for i := range aSlice {
			if !deepEqual(aSlice[i], bSlice[i]) {
				return false
			}
		}
		return true
	}

	// Handle maps
	aMap, aIsMap := a.(map[string]interface{})
	bMap, bIsMap := b.(map[string]interface{})
	if aIsMap && bIsMap {
		if len(aMap) != len(bMap) {
			return false
		}
		for k, v := range aMap {
			if !deepEqual(v, bMap[k]) {
				return false
			}
		}
		return true
	}

	// Default comparison
	return a == b
}
