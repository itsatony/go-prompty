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
	assert.Equal(t, ErrorStrategyDefault, ctx.ErrorStrategy())
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
		})

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
		})

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
		})

		child := parent.Child(map[string]any{
			"cKey": "cValue",
		})

		// Child can see grandparent
		val, ok := child.Get("gpKey")
		assert.True(t, ok)
		assert.Equal(t, "gpValue", val)
	})

	t.Run("child with nil data", func(t *testing.T) {
		parent := NewContext(map[string]any{
			"key": "value",
		})

		child := parent.Child(nil)

		val, ok := child.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("inherits error strategy", func(t *testing.T) {
		parent := NewContextWithStrategy(nil, ErrorStrategyDefault)
		child := parent.Child(nil)

		assert.Equal(t, ErrorStrategyDefault, child.ErrorStrategy())
	})
}

func TestContext_Parent(t *testing.T) {
	parent := NewContext(nil)
	child := parent.Child(nil)

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
	})

	// Should fall back to parent
	val, ok := child.Get("parent.value")
	assert.True(t, ok)
	assert.Equal(t, "fromParent", val)
}
