package prompty

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext_NewContext(t *testing.T) {
	t.Run("with data", func(t *testing.T) {
		data := map[string]any{"key": "value"}
		ctx := NewContext(data)
		require.NotNil(t, ctx)

		val, ok := ctx.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("with nil data", func(t *testing.T) {
		ctx := NewContext(nil)
		require.NotNil(t, ctx)

		_, ok := ctx.Get("key")
		assert.False(t, ok)
	})
}

func TestContext_NewContextWithStrategy(t *testing.T) {
	ctx := NewContextWithStrategy(nil, ErrorStrategyDefault)
	require.NotNil(t, ctx)
	assert.Equal(t, ErrorStrategyDefault, ctx.ErrorStrategyValue())
}

func TestContext_Get(t *testing.T) {
	t.Run("simple key", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"name": "Alice",
			"age":  30,
		})

		val, ok := ctx.Get("name")
		assert.True(t, ok)
		assert.Equal(t, "Alice", val)

		val, ok = ctx.Get("age")
		assert.True(t, ok)
		assert.Equal(t, 30, val)
	})

	t.Run("nested path", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "Bob",
				},
			},
		})

		val, ok := ctx.Get("user.profile.name")
		assert.True(t, ok)
		assert.Equal(t, "Bob", val)
	})

	t.Run("nested path with string map", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"config": map[string]string{
				"env": "production",
			},
		})

		val, ok := ctx.Get("config.env")
		assert.True(t, ok)
		assert.Equal(t, "production", val)
	})

	t.Run("non-existent key", func(t *testing.T) {
		ctx := NewContext(map[string]any{"key": "value"})

		_, ok := ctx.Get("nonexistent")
		assert.False(t, ok)
	})

	t.Run("non-existent nested path", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"user": map[string]any{},
		})

		_, ok := ctx.Get("user.profile.name")
		assert.False(t, ok)
	})

	t.Run("empty path", func(t *testing.T) {
		ctx := NewContext(map[string]any{"key": "value"})

		_, ok := ctx.Get("")
		assert.False(t, ok)
	})

	t.Run("path with empty parts", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"user": map[string]any{
				"name": "Test",
			},
		})

		val, ok := ctx.Get("user..name")
		// Empty parts should be skipped
		assert.True(t, ok)
		assert.Equal(t, "Test", val)
	})

	t.Run("path to non-map", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"name": "Alice",
		})

		// Trying to traverse into a string
		_, ok := ctx.Get("name.first")
		assert.False(t, ok)
	})
}

func TestContext_GetString(t *testing.T) {
	ctx := NewContext(map[string]any{
		"name":   "Alice",
		"number": 42,
	})

	// String value
	assert.Equal(t, "Alice", ctx.GetString("name"))

	// Non-string value
	assert.Equal(t, "", ctx.GetString("number"))

	// Non-existent
	assert.Equal(t, "", ctx.GetString("missing"))
}

func TestContext_GetDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": "value",
	})

	// Existing value
	assert.Equal(t, "value", ctx.GetDefault("existing", "default"))

	// Non-existent value
	assert.Equal(t, "default", ctx.GetDefault("missing", "default"))
}

func TestContext_GetStringDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"name":   "Alice",
		"number": 42,
	})

	// Existing string
	assert.Equal(t, "Alice", ctx.GetStringDefault("name", "default"))

	// Non-string value
	assert.Equal(t, "default", ctx.GetStringDefault("number", "default"))

	// Non-existent
	assert.Equal(t, "default", ctx.GetStringDefault("missing", "default"))
}

func TestContext_Set(t *testing.T) {
	ctx := NewContext(nil)

	ctx.Set("key", "value")

	val, ok := ctx.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestContext_Has(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": "value",
	})

	assert.True(t, ctx.Has("existing"))
	assert.False(t, ctx.Has("missing"))
}

func TestContext_Child(t *testing.T) {
	t.Run("inherits from parent", func(t *testing.T) {
		parent := NewContext(map[string]any{
			"parentKey": "parentValue",
		})

		child := parent.Child(map[string]any{
			"childKey": "childValue",
		}).(*Context)

		// Child has its own data
		val, ok := child.Get("childKey")
		assert.True(t, ok)
		assert.Equal(t, "childValue", val)

		// Child inherits from parent
		val, ok = child.Get("parentKey")
		assert.True(t, ok)
		assert.Equal(t, "parentValue", val)
	})

	t.Run("child can override parent", func(t *testing.T) {
		parent := NewContext(map[string]any{
			"key": "parentValue",
		})

		child := parent.Child(map[string]any{
			"key": "childValue",
		}).(*Context)

		val, ok := child.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "childValue", val)
	})

	t.Run("nested inheritance", func(t *testing.T) {
		grandparent := NewContext(map[string]any{
			"gpKey": "gpValue",
		})

		parent := grandparent.Child(map[string]any{
			"pKey": "pValue",
		}).(*Context)

		child := parent.Child(map[string]any{
			"cKey": "cValue",
		}).(*Context)

		// Child can see grandparent
		val, ok := child.Get("gpKey")
		assert.True(t, ok)
		assert.Equal(t, "gpValue", val)
	})

	t.Run("child with nil data", func(t *testing.T) {
		parent := NewContext(map[string]any{
			"key": "value",
		})

		child := parent.Child(nil).(*Context)

		val, ok := child.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("inherits error strategy", func(t *testing.T) {
		parent := NewContextWithStrategy(nil, ErrorStrategyDefault)
		child := parent.Child(nil).(*Context)

		assert.Equal(t, ErrorStrategyDefault, child.ErrorStrategyValue())
	})
}

func TestContext_Parent(t *testing.T) {
	parent := NewContext(nil)
	child := parent.Child(nil).(*Context)

	assert.Equal(t, parent, child.Parent())
	assert.Nil(t, parent.Parent())
}

func TestContext_Data(t *testing.T) {
	ctx := NewContext(map[string]any{
		"key1": "value1",
		"key2": "value2",
	})

	data := ctx.Data()

	// Should be a copy
	assert.Equal(t, "value1", data["key1"])
	assert.Equal(t, "value2", data["key2"])

	// Modifying copy shouldn't affect original
	data["key1"] = "modified"
	val, _ := ctx.Get("key1")
	assert.Equal(t, "value1", val)
}

func TestContext_ConcurrentAccess(t *testing.T) {
	ctx := NewContext(map[string]any{
		"counter": 0,
	})

	var wg sync.WaitGroup
	const numGoroutines = 100
	const numOps = 100

	// Concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOps; j++ {
				// Mix of operations
				switch j % 3 {
				case 0:
					ctx.Get("counter")
				case 1:
					ctx.Has("counter")
				case 2:
					ctx.Set("counter", j)
				}
			}
		}(i)
	}

	wg.Wait()

	// Context should still be accessible
	assert.True(t, ctx.Has("counter"))
}

func TestContext_NestedPathResolution(t *testing.T) {
	ctx := NewContext(map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"value": "deep",
				},
			},
		},
	})

	val, ok := ctx.Get("level1.level2.level3.value")
	assert.True(t, ok)
	assert.Equal(t, "deep", val)
}

func TestContext_ParentFallback(t *testing.T) {
	parent := NewContext(map[string]any{
		"parent": map[string]any{
			"value": "fromParent",
		},
	})

	child := parent.Child(map[string]any{
		"child": "data",
	}).(*Context)

	// Should fall back to parent
	val, ok := child.Get("parent.value")
	assert.True(t, ok)
	assert.Equal(t, "fromParent", val)
}

func TestContext_DeepCopy(t *testing.T) {
	t.Run("WithEngine creates true deep copy", func(t *testing.T) {
		// Create context with nested data
		original := NewContext(map[string]any{
			"nested": map[string]any{
				"value": "original",
			},
			"slice": []any{"a", "b", "c"},
		})

		// Create copy with engine
		copied := original.WithEngine(nil)

		// Modify the nested map in original
		nestedMap, _ := original.Get("nested")
		nestedMap.(map[string]any)["value"] = "modified"

		// Original should reflect the change
		val, _ := original.Get("nested.value")
		assert.Equal(t, "modified", val)

		// Copy should NOT reflect the change (deep copy)
		val, _ = copied.Get("nested.value")
		assert.Equal(t, "original", val)
	})

	t.Run("WithDepth creates true deep copy", func(t *testing.T) {
		original := NewContext(map[string]any{
			"nested": map[string]any{
				"items": []any{1, 2, 3},
			},
		})

		copied := original.WithDepth(5)

		// Modify original's nested slice
		nestedMap, _ := original.Get("nested")
		items := nestedMap.(map[string]any)["items"].([]any)
		items[0] = 999

		// Original should reflect the change
		val, _ := original.Get("nested")
		assert.Equal(t, 999, val.(map[string]any)["items"].([]any)[0])

		// Copy should NOT reflect the change
		val, _ = copied.Get("nested")
		assert.Equal(t, 1, val.(map[string]any)["items"].([]any)[0])
	})

	t.Run("Data returns deep copy", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"nested": map[string]any{
				"key": "value",
			},
		})

		data := ctx.Data()

		// Modify the copy
		data["nested"].(map[string]any)["key"] = "modified"

		// Original should not be affected
		val, _ := ctx.Get("nested.key")
		assert.Equal(t, "value", val)
	})
}

func TestContext_GetInt(t *testing.T) {
	ctx := NewContext(map[string]any{
		"int":     42,
		"int64":   int64(100),
		"float64": 3.14,
		"string":  "not a number",
		"zero":    0,
	})

	t.Run("int value", func(t *testing.T) {
		val, ok := ctx.GetInt("int")
		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})

	t.Run("int64 value", func(t *testing.T) {
		val, ok := ctx.GetInt("int64")
		assert.True(t, ok)
		assert.Equal(t, 100, val)
	})

	t.Run("float64 value truncates", func(t *testing.T) {
		val, ok := ctx.GetInt("float64")
		assert.True(t, ok)
		assert.Equal(t, 3, val)
	})

	t.Run("string value fails", func(t *testing.T) {
		_, ok := ctx.GetInt("string")
		assert.False(t, ok)
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := ctx.GetInt("missing")
		assert.False(t, ok)
	})

	t.Run("zero value", func(t *testing.T) {
		val, ok := ctx.GetInt("zero")
		assert.True(t, ok)
		assert.Equal(t, 0, val)
	})
}

func TestContext_GetIntDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": 42,
	})

	assert.Equal(t, 42, ctx.GetIntDefault("existing", 0))
	assert.Equal(t, 99, ctx.GetIntDefault("missing", 99))
}

func TestContext_GetFloat(t *testing.T) {
	ctx := NewContext(map[string]any{
		"float64": 3.14,
		"float32": float32(2.5),
		"int":     42,
		"string":  "not a number",
	})

	t.Run("float64 value", func(t *testing.T) {
		val, ok := ctx.GetFloat("float64")
		assert.True(t, ok)
		assert.InDelta(t, 3.14, val, 0.001)
	})

	t.Run("float32 value", func(t *testing.T) {
		val, ok := ctx.GetFloat("float32")
		assert.True(t, ok)
		assert.InDelta(t, 2.5, val, 0.001)
	})

	t.Run("int value converts", func(t *testing.T) {
		val, ok := ctx.GetFloat("int")
		assert.True(t, ok)
		assert.Equal(t, 42.0, val)
	})

	t.Run("string value fails", func(t *testing.T) {
		_, ok := ctx.GetFloat("string")
		assert.False(t, ok)
	})
}

func TestContext_GetFloatDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": 3.14,
	})

	assert.InDelta(t, 3.14, ctx.GetFloatDefault("existing", 0.0), 0.001)
	assert.Equal(t, 9.99, ctx.GetFloatDefault("missing", 9.99))
}

func TestContext_GetBool(t *testing.T) {
	ctx := NewContext(map[string]any{
		"true":    true,
		"false":   false,
		"string":  "true",
		"integer": 1,
	})

	t.Run("true value", func(t *testing.T) {
		val, ok := ctx.GetBool("true")
		assert.True(t, ok)
		assert.True(t, val)
	})

	t.Run("false value", func(t *testing.T) {
		val, ok := ctx.GetBool("false")
		assert.True(t, ok)
		assert.False(t, val)
	})

	t.Run("string value fails", func(t *testing.T) {
		_, ok := ctx.GetBool("string")
		assert.False(t, ok)
	})

	t.Run("integer value fails", func(t *testing.T) {
		_, ok := ctx.GetBool("integer")
		assert.False(t, ok)
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := ctx.GetBool("missing")
		assert.False(t, ok)
	})
}

func TestContext_GetBoolDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": true,
	})

	assert.True(t, ctx.GetBoolDefault("existing", false))
	assert.True(t, ctx.GetBoolDefault("missing", true))
	assert.False(t, ctx.GetBoolDefault("missing", false))
}

func TestContext_GetSlice(t *testing.T) {
	ctx := NewContext(map[string]any{
		"anySlice":    []any{"a", "b", "c"},
		"stringSlice": []string{"x", "y", "z"},
		"intSlice":    []int{1, 2, 3},
		"notSlice":    "just a string",
	})

	t.Run("any slice", func(t *testing.T) {
		val, ok := ctx.GetSlice("anySlice")
		assert.True(t, ok)
		assert.Equal(t, []any{"a", "b", "c"}, val)
	})

	t.Run("string slice converts", func(t *testing.T) {
		val, ok := ctx.GetSlice("stringSlice")
		assert.True(t, ok)
		assert.Len(t, val, 3)
		assert.Equal(t, "x", val[0])
	})

	t.Run("int slice converts", func(t *testing.T) {
		val, ok := ctx.GetSlice("intSlice")
		assert.True(t, ok)
		assert.Len(t, val, 3)
		assert.Equal(t, 1, val[0])
	})

	t.Run("non-slice fails", func(t *testing.T) {
		_, ok := ctx.GetSlice("notSlice")
		assert.False(t, ok)
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := ctx.GetSlice("missing")
		assert.False(t, ok)
	})
}

func TestContext_GetSliceDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": []any{1, 2, 3},
	})

	val := ctx.GetSliceDefault("existing", nil)
	assert.Equal(t, []any{1, 2, 3}, val)

	defaultVal := []any{"default"}
	val = ctx.GetSliceDefault("missing", defaultVal)
	assert.Equal(t, defaultVal, val)
}

func TestContext_GetMap(t *testing.T) {
	ctx := NewContext(map[string]any{
		"anyMap":    map[string]any{"key": "value"},
		"stringMap": map[string]string{"foo": "bar"},
		"notMap":    "just a string",
	})

	t.Run("any map", func(t *testing.T) {
		val, ok := ctx.GetMap("anyMap")
		assert.True(t, ok)
		assert.Equal(t, map[string]any{"key": "value"}, val)
	})

	t.Run("string map converts", func(t *testing.T) {
		val, ok := ctx.GetMap("stringMap")
		assert.True(t, ok)
		assert.Equal(t, "bar", val["foo"])
	})

	t.Run("non-map fails", func(t *testing.T) {
		_, ok := ctx.GetMap("notMap")
		assert.False(t, ok)
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := ctx.GetMap("missing")
		assert.False(t, ok)
	})
}

func TestContext_GetMapDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": map[string]any{"key": "value"},
	})

	val := ctx.GetMapDefault("existing", nil)
	assert.Equal(t, map[string]any{"key": "value"}, val)

	defaultVal := map[string]any{"default": true}
	val = ctx.GetMapDefault("missing", defaultVal)
	assert.Equal(t, defaultVal, val)
}

func TestContext_GetStringSlice(t *testing.T) {
	ctx := NewContext(map[string]any{
		"stringSlice": []string{"a", "b", "c"},
		"anySlice":    []any{"x", "y", "z"},
		"mixedSlice":  []any{"a", 1, "b"},
		"notSlice":    "just a string",
	})

	t.Run("string slice", func(t *testing.T) {
		val, ok := ctx.GetStringSlice("stringSlice")
		assert.True(t, ok)
		assert.Equal(t, []string{"a", "b", "c"}, val)
	})

	t.Run("any slice with all strings converts", func(t *testing.T) {
		val, ok := ctx.GetStringSlice("anySlice")
		assert.True(t, ok)
		assert.Equal(t, []string{"x", "y", "z"}, val)
	})

	t.Run("mixed slice fails", func(t *testing.T) {
		_, ok := ctx.GetStringSlice("mixedSlice")
		assert.False(t, ok)
	})

	t.Run("non-slice fails", func(t *testing.T) {
		_, ok := ctx.GetStringSlice("notSlice")
		assert.False(t, ok)
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := ctx.GetStringSlice("missing")
		assert.False(t, ok)
	})
}

func TestContext_GetStringSliceDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"existing": []string{"a", "b"},
	})

	val := ctx.GetStringSliceDefault("existing", nil)
	assert.Equal(t, []string{"a", "b"}, val)

	defaultVal := []string{"default"}
	val = ctx.GetStringSliceDefault("missing", defaultVal)
	assert.Equal(t, defaultVal, val)
}

func TestDeepCopyValue(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		result := deepCopyValue(nil)
		assert.Nil(t, result)
	})

	t.Run("basic types", func(t *testing.T) {
		assert.Equal(t, "string", deepCopyValue("string"))
		assert.Equal(t, 42, deepCopyValue(42))
		assert.Equal(t, 3.14, deepCopyValue(3.14))
		assert.Equal(t, true, deepCopyValue(true))
	})

	t.Run("map[string]any", func(t *testing.T) {
		original := map[string]any{
			"key": "value",
			"nested": map[string]any{
				"inner": "data",
			},
		}
		copied := deepCopyValue(original).(map[string]any)

		// Modify original
		original["key"] = "modified"
		original["nested"].(map[string]any)["inner"] = "changed"

		// Copy should be unaffected
		assert.Equal(t, "value", copied["key"])
		assert.Equal(t, "data", copied["nested"].(map[string]any)["inner"])
	})

	t.Run("map[string]string", func(t *testing.T) {
		original := map[string]string{"a": "1", "b": "2"}
		copied := deepCopyValue(original).(map[string]string)

		original["a"] = "modified"
		assert.Equal(t, "1", copied["a"])
	})

	t.Run("[]any", func(t *testing.T) {
		original := []any{"a", map[string]any{"key": "value"}}
		copied := deepCopyValue(original).([]any)

		// Modify nested map in original
		original[1].(map[string]any)["key"] = "modified"

		// Copy should be unaffected
		assert.Equal(t, "value", copied[1].(map[string]any)["key"])
	})

	t.Run("[]string", func(t *testing.T) {
		original := []string{"a", "b", "c"}
		copied := deepCopyValue(original).([]string)

		original[0] = "modified"
		assert.Equal(t, "a", copied[0])
	})

	t.Run("[]int", func(t *testing.T) {
		original := []int{1, 2, 3}
		copied := deepCopyValue(original).([]int)

		original[0] = 999
		assert.Equal(t, 1, copied[0])
	})

	t.Run("[]float64", func(t *testing.T) {
		original := []float64{1.1, 2.2, 3.3}
		copied := deepCopyValue(original).([]float64)

		original[0] = 999.0
		assert.InDelta(t, 1.1, copied[0], 0.001)
	})

	t.Run("[]bool", func(t *testing.T) {
		original := []bool{true, false, true}
		copied := deepCopyValue(original).([]bool)

		original[0] = false
		assert.True(t, copied[0])
	})
}

func TestDeepCopyMap(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		result := deepCopyMap(nil)
		assert.Nil(t, result)
	})

	t.Run("deeply nested structure", func(t *testing.T) {
		original := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"value": "deep",
					},
				},
			},
		}

		copied := deepCopyMap(original)

		// Modify deep value in original
		original["level1"].(map[string]any)["level2"].(map[string]any)["level3"].(map[string]any)["value"] = "modified"

		// Copy should be unaffected
		deepVal := copied["level1"].(map[string]any)["level2"].(map[string]any)["level3"].(map[string]any)["value"]
		assert.Equal(t, "deep", deepVal)
	})
}

func TestDeepCopySlice(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		result := deepCopySlice(nil)
		assert.Nil(t, result)
	})

	t.Run("slice with nested maps", func(t *testing.T) {
		original := []any{
			map[string]any{"key": "value1"},
			map[string]any{"key": "value2"},
		}

		copied := deepCopySlice(original)

		// Modify original
		original[0].(map[string]any)["key"] = "modified"

		// Copy should be unaffected
		assert.Equal(t, "value1", copied[0].(map[string]any)["key"])
	})
}

func TestContext_Keys(t *testing.T) {
	t.Run("returns top-level keys", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"name":  "Alice",
			"age":   30,
			"email": "alice@example.com",
		})

		keys := ctx.Keys()
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "name")
		assert.Contains(t, keys, "age")
		assert.Contains(t, keys, "email")
	})

	t.Run("empty context", func(t *testing.T) {
		ctx := NewContext(nil)
		keys := ctx.Keys()
		assert.Empty(t, keys)
	})

	t.Run("does not include parent keys", func(t *testing.T) {
		parent := NewContext(map[string]any{
			"parentKey": "parentValue",
		})
		child := parent.Child(map[string]any{
			"childKey": "childValue",
		}).(*Context)

		// Keys() should only return child's direct keys
		keys := child.Keys()
		assert.Len(t, keys, 1)
		assert.Contains(t, keys, "childKey")
		assert.NotContains(t, keys, "parentKey")
	})
}

func TestContext_AllKeys(t *testing.T) {
	t.Run("includes parent keys", func(t *testing.T) {
		parent := NewContext(map[string]any{
			"parentKey1": "value1",
			"parentKey2": "value2",
		})
		child := parent.Child(map[string]any{
			"childKey": "childValue",
		}).(*Context)

		keys := child.AllKeys()
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "parentKey1")
		assert.Contains(t, keys, "parentKey2")
		assert.Contains(t, keys, "childKey")
	})

	t.Run("child keys take precedence over parent", func(t *testing.T) {
		parent := NewContext(map[string]any{
			"key": "parentValue",
		})
		child := parent.Child(map[string]any{
			"key": "childValue",
		}).(*Context)

		keys := child.AllKeys()
		// Should only contain one "key" (deduplicated)
		count := 0
		for _, k := range keys {
			if k == "key" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("empty context", func(t *testing.T) {
		ctx := NewContext(nil)
		keys := ctx.AllKeys()
		assert.Empty(t, keys)
	})

	t.Run("deep hierarchy", func(t *testing.T) {
		grandparent := NewContext(map[string]any{"gpKey": "value"})
		parent := grandparent.Child(map[string]any{"pKey": "value"}).(*Context)
		child := parent.Child(map[string]any{"cKey": "value"}).(*Context)

		keys := child.AllKeys()
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "gpKey")
		assert.Contains(t, keys, "pKey")
		assert.Contains(t, keys, "cKey")
	})
}
