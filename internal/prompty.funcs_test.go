package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncRegistry_Register(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 1,
		Fn:      func(args []any) (any, error) { return args[0], nil },
	}

	err := r.Register(fn)
	require.NoError(t, err)

	assert.True(t, r.Has("test"))
}

func TestFuncRegistry_Register_Duplicate(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 1,
		Fn:      func(args []any) (any, error) { return args[0], nil },
	}

	err := r.Register(fn)
	require.NoError(t, err)

	// Second registration should fail
	err = r.Register(fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFuncAlreadyExists)
}

func TestFuncRegistry_MustRegister(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 1,
		Fn:      func(args []any) (any, error) { return args[0], nil },
	}

	assert.NotPanics(t, func() {
		r.MustRegister(fn)
	})
}

func TestFuncRegistry_MustRegister_Panic(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 1,
		Fn:      func(args []any) (any, error) { return args[0], nil },
	}

	r.MustRegister(fn)

	assert.Panics(t, func() {
		r.MustRegister(fn) // duplicate
	})
}

func TestFuncRegistry_Get(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 1,
		Fn:      func(args []any) (any, error) { return args[0], nil },
	}

	r.MustRegister(fn)

	retrieved, ok := r.Get("test")
	require.True(t, ok)
	assert.Equal(t, fn, retrieved)

	_, ok = r.Get("nonexistent")
	assert.False(t, ok)
}

func TestFuncRegistry_Has(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 1,
		Fn:      func(args []any) (any, error) { return args[0], nil },
	}

	assert.False(t, r.Has("test"))
	r.MustRegister(fn)
	assert.True(t, r.Has("test"))
}

func TestFuncRegistry_Call(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "double",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			if n, ok := args[0].(float64); ok {
				return n * 2, nil
			}
			return nil, nil
		},
	}

	r.MustRegister(fn)

	result, err := r.Call("double", []any{float64(5)})
	require.NoError(t, err)
	assert.Equal(t, float64(10), result)
}

func TestFuncRegistry_Call_NotFound(t *testing.T) {
	r := NewFuncRegistry()

	_, err := r.Call("nonexistent", []any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFuncNotFound)
}

func TestFuncRegistry_Call_TooFewArgs(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "needsTwo",
		MinArgs: 2,
		MaxArgs: 2,
		Fn:      func(args []any) (any, error) { return nil, nil },
	}

	r.MustRegister(fn)

	_, err := r.Call("needsTwo", []any{"one"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFuncTooFewArgs)
}

func TestFuncRegistry_Call_TooManyArgs(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "needsOne",
		MinArgs: 1,
		MaxArgs: 1,
		Fn:      func(args []any) (any, error) { return nil, nil },
	}

	r.MustRegister(fn)

	_, err := r.Call("needsOne", []any{"one", "two"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFuncTooManyArgs)
}

func TestFuncRegistry_Call_VariadicArgs(t *testing.T) {
	r := NewFuncRegistry()

	fn := &Func{
		Name:    "sum",
		MinArgs: 1,
		MaxArgs: -1, // unlimited
		Fn: func(args []any) (any, error) {
			sum := 0.0
			for _, a := range args {
				if n, ok := a.(float64); ok {
					sum += n
				}
			}
			return sum, nil
		},
	}

	r.MustRegister(fn)

	result, err := r.Call("sum", []any{float64(1), float64(2), float64(3)})
	require.NoError(t, err)
	assert.Equal(t, float64(6), result)
}

func TestFuncRegistry_List(t *testing.T) {
	r := NewFuncRegistry()

	r.MustRegister(&Func{Name: "b", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	r.MustRegister(&Func{Name: "a", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	r.MustRegister(&Func{Name: "c", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})

	list := r.List()
	assert.Len(t, list, 3)
	assert.Contains(t, list, "a")
	assert.Contains(t, list, "b")
	assert.Contains(t, list, "c")
}

func TestRegisterBuiltinFuncs(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	// Check that expected functions are registered
	expectedFuncs := []string{
		"len", "upper", "lower", "trim", "contains",
		"hasPrefix", "hasSuffix", "split", "join",
		"first", "last", "keys", "values", "has",
		"toString", "toInt", "toFloat", "toBool",
		"typeOf", "isNil", "isEmpty",
		"default", "coalesce",
		"trimPrefix", "trimSuffix", "replace",
	}

	for _, name := range expectedFuncs {
		assert.True(t, r.Has(name), "expected function %s to be registered", name)
	}
}

// String function tests
func TestBuiltinFunc_Upper(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("upper", []any{"hello"})
	require.NoError(t, err)
	assert.Equal(t, "HELLO", result)
}

func TestBuiltinFunc_Lower(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("lower", []any{"HELLO"})
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestBuiltinFunc_Trim(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("trim", []any{"  hello  "})
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestBuiltinFunc_TrimPrefix(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("trimPrefix", []any{"hello world", "hello "})
	require.NoError(t, err)
	assert.Equal(t, "world", result)
}

func TestBuiltinFunc_TrimSuffix(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("trimSuffix", []any{"hello world", " world"})
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestBuiltinFunc_HasPrefix(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("hasPrefix", []any{"hello world", "hello"})
	require.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = r.Call("hasPrefix", []any{"hello world", "world"})
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestBuiltinFunc_HasSuffix(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("hasSuffix", []any{"hello world", "world"})
	require.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = r.Call("hasSuffix", []any{"hello world", "hello"})
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestBuiltinFunc_Contains(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	t.Run("string contains", func(t *testing.T) {
		result, err := r.Call("contains", []any{"hello world", "wor"})
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("array contains", func(t *testing.T) {
		result, err := r.Call("contains", []any{[]any{"a", "b", "c"}, "b"})
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("array does not contain", func(t *testing.T) {
		result, err := r.Call("contains", []any{[]any{"a", "b", "c"}, "d"})
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

func TestBuiltinFunc_Replace(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("replace", []any{"hello world", "world", "universe"})
	require.NoError(t, err)
	assert.Equal(t, "hello universe", result)
}

func TestBuiltinFunc_Split(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("split", []any{"a,b,c", ","})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestBuiltinFunc_Join(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("join", []any{[]any{"a", "b", "c"}, ","})
	require.NoError(t, err)
	assert.Equal(t, "a,b,c", result)
}

// Collection function tests
func TestBuiltinFunc_Len(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"string", "hello", 5},
		{"array", []any{1, 2, 3}, 3},
		{"map", map[string]any{"a": 1, "b": 2}, 2},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call("len", []any{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_First(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("first", []any{[]any{"a", "b", "c"}})
	require.NoError(t, err)
	assert.Equal(t, "a", result)

	result, err = r.Call("first", []any{[]any{}})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestBuiltinFunc_Last(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("last", []any{[]any{"a", "b", "c"}})
	require.NoError(t, err)
	assert.Equal(t, "c", result)

	result, err = r.Call("last", []any{[]any{}})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestBuiltinFunc_Keys(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("keys", []any{map[string]any{"b": 1, "a": 2}})
	require.NoError(t, err)
	// Keys should be sorted
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestBuiltinFunc_Values(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("values", []any{map[string]any{"a": 1, "b": 2}})
	require.NoError(t, err)
	// Values should be in key-sorted order
	values := result.([]any)
	assert.Len(t, values, 2)
}

func TestBuiltinFunc_Has(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("has", []any{map[string]any{"a": 1}, "a"})
	require.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = r.Call("has", []any{map[string]any{"a": 1}, "b"})
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

// Type function tests
func TestBuiltinFunc_ToString(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call("toString", []any{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_ToInt(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"int", 42, 42},
		{"float", 3.14, 3},
		{"string", "42", 42},
		{"bool true", true, 1},
		{"bool false", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call("toInt", []any{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_ToFloat(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"int", 42, 42.0},
		{"float", 3.14, 3.14},
		{"string", "3.14", 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call("toFloat", []any{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_ToBool(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string false - truthy non-empty string", "false", true}, // "false" is a non-empty string, so truthy
		{"non-empty string", "hello", true},
		{"empty string", "", false},
		{"positive number", 1, true},
		{"zero", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call("toBool", []any{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_TypeOf(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "string"},
		{"int", 42, "int"},
		{"float", 3.14, "float64"},
		{"bool", true, "bool"},
		{"nil", nil, "nil"},
		{"array", []any{1, 2}, "[]interface {}"},
		{"map", map[string]any{"a": 1}, "map[string]interface {}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call("typeOf", []any{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_IsNil(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	result, err := r.Call("isNil", []any{nil})
	require.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = r.Call("isNil", []any{"hello"})
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestBuiltinFunc_IsEmpty(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"empty array", []any{}, true},
		{"empty map", map[string]any{}, true},
		{"non-empty string", "hello", false},
		{"non-empty array", []any{1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call("isEmpty", []any{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: isTruthy is not a registered function, it's an internal helper.
// The toBool function uses isTruthy internally, so we test truthiness via toBool.

// Utility function tests
func TestBuiltinFunc_Default(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	t.Run("returns value if not nil", func(t *testing.T) {
		result, err := r.Call("default", []any{"hello", "fallback"})
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("returns default if nil", func(t *testing.T) {
		result, err := r.Call("default", []any{nil, "fallback"})
		require.NoError(t, err)
		assert.Equal(t, "fallback", result)
	})

	t.Run("returns default if empty string", func(t *testing.T) {
		result, err := r.Call("default", []any{"", "fallback"})
		require.NoError(t, err)
		assert.Equal(t, "fallback", result)
	})
}

func TestBuiltinFunc_Coalesce(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	t.Run("returns first non-nil", func(t *testing.T) {
		result, err := r.Call("coalesce", []any{nil, "", "hello", "world"})
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("returns nil if all nil or empty", func(t *testing.T) {
		result, err := r.Call("coalesce", []any{nil, "", nil})
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}
