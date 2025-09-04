package starlark

import (
	"math"
	"testing"

	"go.starlark.net/starlark"
)

func TestGoToStarlarkValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType string
		wantErr  bool
	}{
		{"nil", nil, "NoneType", false},
		{"bool true", true, "bool", false},
		{"bool false", false, "bool", false},
		{"int", 42, "int", false},
		{"int64", int64(123456789), "int", false},
		{"float64", 3.14, "float", false},
		{"string", "hello", "string", false},
		{"empty string", "", "string", false},
		{"empty slice", []interface{}{}, "list", false},
		{"slice with values", []interface{}{1, "two", 3.0}, "list", false},
		{"empty map", map[string]interface{}{}, "dict", false},
		{"map with values", map[string]interface{}{"key": "value", "num": 42}, "dict", false},
		{"nested slice", []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}}, "list", false},
		{"nested map", map[string]interface{}{"nested": map[string]interface{}{"inner": "value"}}, "dict", false},
		{"unsupported type", make(chan int), "", true},
		{"function type", func() {}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GoToStarlarkValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GoToStarlarkValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotType := got.Type()
				if gotType != tt.wantType {
					t.Errorf("GoToStarlarkValue() type = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}

func TestGoToStarlarkValue_SpecialFloats(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{"positive infinity", math.Inf(1)},
		{"negative infinity", math.Inf(-1)},
		{"NaN", math.NaN()},
		{"very large number", math.MaxFloat64},
		{"very small number", math.SmallestNonzeroFloat64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GoToStarlarkValue(tt.input)
			if err != nil {
				t.Errorf("GoToStarlarkValue() error = %v", err)
				return
			}
			if got.Type() != "float" {
				t.Errorf("GoToStarlarkValue() type = %v, want float", got.Type())
			}
		})
	}
}

func TestStarlarkToGoValue(t *testing.T) {
	tests := []struct {
		name    string
		input   starlark.Value
		want    interface{}
		wantErr bool
	}{
		{"None", starlark.None, nil, false},
		{"Bool true", starlark.True, true, false},
		{"Bool false", starlark.False, false, false},
		{"small int", starlark.MakeInt(42), int64(42), false},
		{"large int", starlark.MakeUint64(math.MaxUint64), "18446744073709551615", false},
		{"float", starlark.Float(3.14), 3.14, false},
		{"string", starlark.String("hello"), "hello", false},
		{"empty string", starlark.String(""), "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StarlarkToGoValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("StarlarkToGoValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("StarlarkToGoValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStarlarkToGoValue_List(t *testing.T) {
	// Create a Starlark list with mixed types
	list := starlark.NewList([]starlark.Value{
		starlark.MakeInt(1),
		starlark.String("two"),
		starlark.Float(3.0),
		starlark.None,
	})

	got, err := StarlarkToGoValue(list)
	if err != nil {
		t.Errorf("StarlarkToGoValue() error = %v", err)
		return
	}

	want := []interface{}{int64(1), "two", 3.0, nil}
	gotSlice, ok := got.([]interface{})
	if !ok {
		t.Errorf("StarlarkToGoValue() type = %T, want []interface{}", got)
		return
	}

	if len(gotSlice) != len(want) {
		t.Errorf("StarlarkToGoValue() length = %d, want %d", len(gotSlice), len(want))
		return
	}

	for i, v := range want {
		if gotSlice[i] != v {
			t.Errorf("StarlarkToGoValue()[%d] = %v, want %v", i, gotSlice[i], v)
		}
	}
}

func TestStarlarkToGoValue_Dict(t *testing.T) {
	// Create a Starlark dict with mixed types
	dict := starlark.NewDict(2)
	dict.SetKey(starlark.String("string_key"), starlark.String("value"))
	dict.SetKey(starlark.String("int_key"), starlark.MakeInt(42))
	dict.SetKey(starlark.MakeInt(123), starlark.String("non-string key"))

	got, err := StarlarkToGoValue(dict)
	if err != nil {
		t.Errorf("StarlarkToGoValue() error = %v", err)
		return
	}

	gotMap, ok := got.(map[string]interface{})
	if !ok {
		t.Errorf("StarlarkToGoValue() type = %T, want map[string]interface{}", got)
		return
	}

	// Should only contain string keys
	if len(gotMap) != 2 {
		t.Errorf("StarlarkToGoValue() map length = %d, want 2", len(gotMap))
	}

	if gotMap["string_key"] != "value" {
		t.Errorf("StarlarkToGoValue()[string_key] = %v, want 'value'", gotMap["string_key"])
	}

	if gotMap["int_key"] != int64(42) {
		t.Errorf("StarlarkToGoValue()[int_key] = %v, want 42", gotMap["int_key"])
	}

	// Non-string key should be skipped
	if _, exists := gotMap["123"]; exists {
		t.Errorf("StarlarkToGoValue() should skip non-string keys")
	}
}

func TestStarlarkToGoValue_NestedStructures(t *testing.T) {
	// Create nested list: [[1, 2], [3, 4]]
	innerList1 := starlark.NewList([]starlark.Value{
		starlark.MakeInt(1),
		starlark.MakeInt(2),
	})
	innerList2 := starlark.NewList([]starlark.Value{
		starlark.MakeInt(3),
		starlark.MakeInt(4),
	})
	outerList := starlark.NewList([]starlark.Value{innerList1, innerList2})

	got, err := StarlarkToGoValue(outerList)
	if err != nil {
		t.Errorf("StarlarkToGoValue() error = %v", err)
		return
	}

	want := []interface{}{
		[]interface{}{int64(1), int64(2)},
		[]interface{}{int64(3), int64(4)},
	}

	gotSlice := got.([]interface{})
	for i, innerWant := range want {
		innerGot := gotSlice[i].([]interface{})
		innerWantSlice := innerWant.([]interface{})
		for j, v := range innerWantSlice {
			if innerGot[j] != v {
				t.Errorf("StarlarkToGoValue()[%d][%d] = %v, want %v", i, j, innerGot[j], v)
			}
		}
	}
}

func TestRoundTripConversion(t *testing.T) {
	testCases := []interface{}{
		nil,
		true,
		false,
		42,
		int64(123456789),
		3.14159,
		"hello world",
		[]interface{}{1, "two", 3.0},
		map[string]interface{}{"key": "value", "number": 42},
		map[string]interface{}{
			"nested": map[string]interface{}{
				"inner": []interface{}{1, 2, 3},
			},
		},
	}

	for i, original := range testCases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			// Go -> Starlark
			starVal, err := GoToStarlarkValue(original)
			if err != nil {
				t.Errorf("GoToStarlarkValue() error = %v", err)
				return
			}

			// Starlark -> Go
			result, err := StarlarkToGoValue(starVal)
			if err != nil {
				t.Errorf("StarlarkToGoValue() error = %v", err)
				return
			}

			// Compare (with type adjustments for ints)
			if !equalValues(original, result) {
				t.Errorf("Round trip failed: original = %v (%T), result = %v (%T)", original, original, result, result)
			}
		})
	}
}

// equalValues compares two values, handling int/int64 differences
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