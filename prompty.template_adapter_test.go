package prompty

import (
	"testing"

	"github.com/itsatony/go-prompty/v2/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalAttributesAdapter_Get(t *testing.T) {
	t.Run("existing key returns value and true", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"name": "Alice", "role": "admin"},
		}

		val, ok := adapter.Get("name")
		assert.True(t, ok)
		assert.Equal(t, "Alice", val)

		val, ok = adapter.Get("role")
		assert.True(t, ok)
		assert.Equal(t, "admin", val)
	})

	t.Run("missing key returns empty string and false", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"name": "Alice"},
		}

		val, ok := adapter.Get("missing")
		assert.False(t, ok)
		assert.Equal(t, "", val)
	})

	t.Run("empty attributes returns empty string and false", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{},
		}

		val, ok := adapter.Get("anything")
		assert.False(t, ok)
		assert.Equal(t, "", val)
	})

	t.Run("nil attributes returns empty string and false", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: nil,
		}

		val, ok := adapter.Get("anything")
		assert.False(t, ok)
		assert.Equal(t, "", val)
	})

	t.Run("empty string key", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"": "empty-key-value"},
		}

		val, ok := adapter.Get("")
		assert.True(t, ok)
		assert.Equal(t, "empty-key-value", val)
	})

	t.Run("empty string value", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"key": ""},
		}

		val, ok := adapter.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "", val)
	})
}

func TestInternalAttributesAdapter_GetDefault(t *testing.T) {
	t.Run("existing key returns value not default", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"name": "Alice"},
		}

		result := adapter.GetDefault("name", "DefaultName")
		assert.Equal(t, "Alice", result)
	})

	t.Run("missing key returns default", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"name": "Alice"},
		}

		result := adapter.GetDefault("missing", "fallback")
		assert.Equal(t, "fallback", result)
	})

	t.Run("empty attributes returns default", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{},
		}

		result := adapter.GetDefault("key", "default-val")
		assert.Equal(t, "default-val", result)
	})

	t.Run("nil attributes returns default", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: nil,
		}

		result := adapter.GetDefault("key", "default-val")
		assert.Equal(t, "default-val", result)
	})

	t.Run("existing key with empty value returns empty string not default", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"key": ""},
		}

		result := adapter.GetDefault("key", "should-not-use")
		assert.Equal(t, "", result)
	})

	t.Run("empty default value", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{},
		}

		result := adapter.GetDefault("missing", "")
		assert.Equal(t, "", result)
	})
}

func TestInternalAttributesAdapter_Has(t *testing.T) {
	t.Run("existing key returns true", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"name": "Alice", "role": "admin"},
		}

		assert.True(t, adapter.Has("name"))
		assert.True(t, adapter.Has("role"))
	})

	t.Run("missing key returns false", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"name": "Alice"},
		}

		assert.False(t, adapter.Has("missing"))
	})

	t.Run("empty attributes returns false", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{},
		}

		assert.False(t, adapter.Has("anything"))
	})

	t.Run("nil attributes returns false", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: nil,
		}

		assert.False(t, adapter.Has("anything"))
	})

	t.Run("key with empty value still returns true", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"key": ""},
		}

		assert.True(t, adapter.Has("key"))
	})
}

func TestInternalAttributesAdapter_Keys(t *testing.T) {
	t.Run("returns all keys in sorted order", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"zebra": "z", "alpha": "a", "middle": "m"},
		}

		keys := adapter.Keys()
		require.Len(t, keys, 3)
		assert.Equal(t, []string{"alpha", "middle", "zebra"}, keys)
	})

	t.Run("single key", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"only": "one"},
		}

		keys := adapter.Keys()
		require.Len(t, keys, 1)
		assert.Equal(t, []string{"only"}, keys)
	})

	t.Run("empty attributes returns empty slice", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{},
		}

		keys := adapter.Keys()
		assert.Empty(t, keys)
	})

	t.Run("nil attributes returns nil", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: nil,
		}

		keys := adapter.Keys()
		assert.Nil(t, keys)
	})
}

func TestInternalAttributesAdapter_Map(t *testing.T) {
	t.Run("returns copy of underlying map", func(t *testing.T) {
		original := internal.Attributes{"key1": "val1", "key2": "val2"}
		adapter := &internalAttributesAdapter{
			attrs: original,
		}

		result := adapter.Map()
		require.Len(t, result, 2)
		assert.Equal(t, "val1", result["key1"])
		assert.Equal(t, "val2", result["key2"])

		// Verify it is a copy by mutating the result and checking the original
		result["key1"] = "mutated"
		origVal, ok := adapter.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "val1", origVal, "modifying returned map must not affect the adapter")
	})

	t.Run("empty attributes returns empty map", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{},
		}

		result := adapter.Map()
		require.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("nil attributes returns empty map", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: nil,
		}

		result := adapter.Map()
		require.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("returned map has correct type", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"a": "1"},
		}

		result := adapter.Map()
		assert.IsType(t, map[string]string{}, result)
	})
}

func TestInternalAttributesAdapter_Integration(t *testing.T) {
	t.Run("all methods consistent on same adapter", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"name": "Alice", "role": "admin"},
		}

		// Has and Get should be consistent
		for _, key := range adapter.Keys() {
			assert.True(t, adapter.Has(key), "Has should return true for key from Keys()")
			val, ok := adapter.Get(key)
			assert.True(t, ok, "Get should return ok=true for key from Keys()")
			assert.NotEmpty(t, val)
		}

		// Map should contain all keys
		m := adapter.Map()
		for _, key := range adapter.Keys() {
			_, exists := m[key]
			assert.True(t, exists, "Map should contain key %q from Keys()", key)
		}

		// Keys length should match Map length
		assert.Equal(t, len(adapter.Keys()), len(adapter.Map()))
	})

	t.Run("GetDefault returns same as Get for existing keys", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{"x": "hello", "y": "world"},
		}

		for _, key := range adapter.Keys() {
			getVal, _ := adapter.Get(key)
			defaultVal := adapter.GetDefault(key, "SHOULD_NOT_APPEAR")
			assert.Equal(t, getVal, defaultVal, "GetDefault should return same value as Get for key %q", key)
		}
	})

	t.Run("special characters in keys and values", func(t *testing.T) {
		adapter := &internalAttributesAdapter{
			attrs: internal.Attributes{
				"key-with-dashes":      "value1",
				"key_with_underscores": "value2",
				"key.with.dots":        "value3",
				"key with spaces":      "value with spaces",
				"unicode-key":          "unicode-value",
			},
		}

		val, ok := adapter.Get("key-with-dashes")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)

		val, ok = adapter.Get("key with spaces")
		assert.True(t, ok)
		assert.Equal(t, "value with spaces", val)

		assert.True(t, adapter.Has("key.with.dots"))
		assert.Equal(t, "value2", adapter.GetDefault("key_with_underscores", ""))

		keys := adapter.Keys()
		assert.Len(t, keys, 5)
	})
}
