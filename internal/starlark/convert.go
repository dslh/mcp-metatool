package starlark

import (
	"fmt"

	"go.starlark.net/starlark"
)

// GoToStarlarkValue converts a Go value to a Starlark value
func GoToStarlarkValue(v interface{}) (starlark.Value, error) {
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
			starVal, err := GoToStarlarkValue(item)
			if err != nil {
				return nil, err
			}
			list.SetIndex(i, starVal)
		}
		return list, nil
	case map[string]interface{}:
		dict := starlark.NewDict(len(val))
		for k, item := range val {
			starVal, err := GoToStarlarkValue(item)
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

// StarlarkToGoValue converts a Starlark value to a Go value
func StarlarkToGoValue(v starlark.Value) (interface{}, error) {
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
			item, err := StarlarkToGoValue(val.Index(i))
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
			goVal, err := StarlarkToGoValue(v)
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