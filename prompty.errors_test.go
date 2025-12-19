package prompty

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/itsatony/go-cuserr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewParseError tests parse error creation with position context
func TestNewParseError(t *testing.T) {
	t.Run("with cause error", func(t *testing.T) {
		pos := Position{Line: 5, Column: 10, Offset: 50}
		causeErr := errors.New("underlying parse issue")
		err := NewParseError(ErrMsgUnexpectedChar, pos, causeErr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgUnexpectedChar)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, strconv.Itoa(pos.Line), line)

		column, ok := customErr.GetMetadata(MetaKeyColumn)
		assert.True(t, ok)
		assert.Equal(t, strconv.Itoa(pos.Column), column)

		offset, ok := customErr.GetMetadata(MetaKeyOffset)
		assert.True(t, ok)
		assert.Equal(t, strconv.Itoa(pos.Offset), offset)

		// Verify error wrapping
		assert.True(t, errors.Is(err, causeErr))
	})

	t.Run("without cause error", func(t *testing.T) {
		pos := Position{Line: 1, Column: 1, Offset: 0}
		err := NewParseError(ErrMsgInvalidSyntax, pos, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgInvalidSyntax)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "1", line)
	})

	t.Run("with zero position", func(t *testing.T) {
		pos := Position{Line: 0, Column: 0, Offset: 0}
		err := NewParseError(ErrMsgParseFailed, pos, nil)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "0", line)
	})
}

// TestNewUnterminatedTagError tests unterminated tag error creation
func TestNewUnterminatedTagError(t *testing.T) {
	t.Run("basic unterminated tag", func(t *testing.T) {
		pos := Position{Line: 10, Column: 25, Offset: 150}
		err := NewUnterminatedTagError(pos)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgUnterminatedTag)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "10", line)

		column, ok := customErr.GetMetadata(MetaKeyColumn)
		assert.True(t, ok)
		assert.Equal(t, "25", column)

		offset, ok := customErr.GetMetadata(MetaKeyOffset)
		assert.True(t, ok)
		assert.Equal(t, "150", offset)
	})

	t.Run("at start of file", func(t *testing.T) {
		pos := Position{Line: 1, Column: 1, Offset: 0}
		err := NewUnterminatedTagError(pos)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgUnterminatedTag)
	})
}

// TestNewUnterminatedStrError tests unterminated string literal error creation
func TestNewUnterminatedStrError(t *testing.T) {
	t.Run("basic unterminated string", func(t *testing.T) {
		pos := Position{Line: 3, Column: 15, Offset: 75}
		err := NewUnterminatedStrError(pos)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgUnterminatedStr)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "3", line)

		column, ok := customErr.GetMetadata(MetaKeyColumn)
		assert.True(t, ok)
		assert.Equal(t, "15", column)

		offset, ok := customErr.GetMetadata(MetaKeyOffset)
		assert.True(t, ok)
		assert.Equal(t, "75", offset)
	})
}

// TestNewMismatchedTagError tests mismatched closing tag error creation
func TestNewMismatchedTagError(t *testing.T) {
	t.Run("basic mismatched tags", func(t *testing.T) {
		pos := Position{Line: 8, Column: 20, Offset: 120}
		err := NewMismatchedTagError("prompty.if", "prompty.for", pos)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgMismatchedTag)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "8", line)

		column, ok := customErr.GetMetadata(MetaKeyColumn)
		assert.True(t, ok)
		assert.Equal(t, "20", column)

		expected, ok := customErr.GetMetadata(MetaKeyExpected)
		assert.True(t, ok)
		assert.Equal(t, "prompty.if", expected)

		actual, ok := customErr.GetMetadata(MetaKeyActual)
		assert.True(t, ok)
		assert.Equal(t, "prompty.for", actual)
	})

	t.Run("with empty tag names", func(t *testing.T) {
		pos := Position{Line: 1, Column: 1, Offset: 0}
		err := NewMismatchedTagError("", "", pos)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		expected, ok := customErr.GetMetadata(MetaKeyExpected)
		assert.True(t, ok)
		assert.Equal(t, "", expected)
	})
}

// TestNewNestedRawBlockError tests nested raw block error creation
func TestNewNestedRawBlockError(t *testing.T) {
	t.Run("basic nested raw block", func(t *testing.T) {
		pos := Position{Line: 12, Column: 5, Offset: 200}
		err := NewNestedRawBlockError(pos)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgNestedRawBlock)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "12", line)

		column, ok := customErr.GetMetadata(MetaKeyColumn)
		assert.True(t, ok)
		assert.Equal(t, "5", column)

		offset, ok := customErr.GetMetadata(MetaKeyOffset)
		assert.True(t, ok)
		assert.Equal(t, "200", offset)
	})
}

// TestNewExecutionError tests execution error creation with tag context
func TestNewExecutionError(t *testing.T) {
	t.Run("with cause error", func(t *testing.T) {
		pos := Position{Line: 7, Column: 12, Offset: 100}
		causeErr := errors.New("resolver failed")
		err := NewExecutionError(ErrMsgResolverFailed, "UserProfile", pos, causeErr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgResolverFailed)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, "UserProfile", tag)

		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "7", line)

		column, ok := customErr.GetMetadata(MetaKeyColumn)
		assert.True(t, ok)
		assert.Equal(t, "12", column)

		// Verify error wrapping
		assert.True(t, errors.Is(err, causeErr))
	})

	t.Run("without cause error", func(t *testing.T) {
		pos := Position{Line: 1, Column: 1, Offset: 0}
		err := NewExecutionError(ErrMsgExecutionFailed, "prompty.var", pos, nil)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, "prompty.var", tag)
	})
}

// TestNewVariableNotFoundError tests variable not found error creation
func TestNewVariableNotFoundError(t *testing.T) {
	t.Run("basic variable not found", func(t *testing.T) {
		err := NewVariableNotFoundError("user.name")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgVariableNotFound)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		path, ok := customErr.GetMetadata(MetaKeyPath)
		assert.True(t, ok)
		assert.Equal(t, "user.name", path)
	})

	t.Run("with nested path", func(t *testing.T) {
		err := NewVariableNotFoundError("user.profile.settings.theme")

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		path, ok := customErr.GetMetadata(MetaKeyPath)
		assert.True(t, ok)
		assert.Equal(t, "user.profile.settings.theme", path)
	})

	t.Run("with empty path", func(t *testing.T) {
		err := NewVariableNotFoundError("")

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		path, ok := customErr.GetMetadata(MetaKeyPath)
		assert.True(t, ok)
		assert.Equal(t, "", path)
	})
}

// TestNewUnknownTagError tests unknown tag error creation
func TestNewUnknownTagError(t *testing.T) {
	t.Run("basic unknown tag", func(t *testing.T) {
		pos := Position{Line: 6, Column: 8, Offset: 80}
		err := NewUnknownTagError("CustomTag", pos)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgUnknownResolver)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, "CustomTag", tag)

		line, ok := customErr.GetMetadata(MetaKeyLine)
		assert.True(t, ok)
		assert.Equal(t, "6", line)

		column, ok := customErr.GetMetadata(MetaKeyColumn)
		assert.True(t, ok)
		assert.Equal(t, "8", column)
	})

	t.Run("with empty tag name", func(t *testing.T) {
		pos := Position{Line: 1, Column: 1, Offset: 0}
		err := NewUnknownTagError("", pos)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, "", tag)
	})
}

// TestNewResolverExistsError tests resolver collision error creation
func TestNewResolverExistsError(t *testing.T) {
	t.Run("basic resolver exists", func(t *testing.T) {
		err := NewResolverExistsError("UserProfile")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgResolverExists)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, "UserProfile", tag)
	})

	t.Run("with builtin tag name", func(t *testing.T) {
		err := NewResolverExistsError(TagNameVar)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, TagNameVar, tag)
	})
}

// TestNewMissingAttributeError tests missing required attribute error creation
func TestNewMissingAttributeError(t *testing.T) {
	t.Run("basic missing attribute", func(t *testing.T) {
		err := NewMissingAttributeError("name", "prompty.var")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgMissingAttribute)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		attr, ok := customErr.GetMetadata(MetaKeyAttribute)
		assert.True(t, ok)
		assert.Equal(t, "name", attr)

		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, "prompty.var", tag)
	})

	t.Run("with custom tag", func(t *testing.T) {
		err := NewMissingAttributeError("id", "UserProfile")

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		attr, ok := customErr.GetMetadata(MetaKeyAttribute)
		assert.True(t, ok)
		assert.Equal(t, "id", attr)
	})
}

// TestNewInvalidAttributeError tests invalid attribute value error creation
func TestNewInvalidAttributeError(t *testing.T) {
	t.Run("basic invalid attribute", func(t *testing.T) {
		err := NewInvalidAttributeError("limit", "abc", "must be a number")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgInvalidAttribute)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		attr, ok := customErr.GetMetadata(MetaKeyAttribute)
		assert.True(t, ok)
		assert.Equal(t, "limit", attr)

		value, ok := customErr.GetMetadata(MetaKeyValue)
		assert.True(t, ok)
		assert.Equal(t, "abc", value)

		reason, ok := customErr.GetMetadata(MetaKeyReason)
		assert.True(t, ok)
		assert.Equal(t, "must be a number", reason)
	})

	t.Run("with empty reason", func(t *testing.T) {
		err := NewInvalidAttributeError("onerror", "invalid", "")

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		reason, ok := customErr.GetMetadata(MetaKeyReason)
		assert.True(t, ok)
		assert.Equal(t, "", reason)
	})
}

// TestNewResolverError tests resolver failure error creation
func TestNewResolverError(t *testing.T) {
	t.Run("with cause error", func(t *testing.T) {
		causeErr := errors.New("database connection failed")
		err := NewResolverError("UserProfile", causeErr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgResolverFailed)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		resolver, ok := customErr.GetMetadata(MetaKeyResolver)
		assert.True(t, ok)
		assert.Equal(t, "UserProfile", resolver)

		// Verify error wrapping
		assert.True(t, errors.Is(err, causeErr))
	})

	t.Run("with nested error", func(t *testing.T) {
		innerErr := fmt.Errorf("timeout: %w", errors.New("connection timeout"))
		err := NewResolverError("DataResolver", innerErr)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		resolver, ok := customErr.GetMetadata(MetaKeyResolver)
		assert.True(t, ok)
		assert.Equal(t, "DataResolver", resolver)
	})
}

// TestNewTypeConversionError tests type conversion error creation
func TestNewTypeConversionError(t *testing.T) {
	t.Run("basic type conversion", func(t *testing.T) {
		err := NewTypeConversionError("string", "int", "abc")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgTypeConversion)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		fromType, ok := customErr.GetMetadata(MetaKeyFromType)
		assert.True(t, ok)
		assert.Equal(t, "string", fromType)

		toType, ok := customErr.GetMetadata(MetaKeyToType)
		assert.True(t, ok)
		assert.Equal(t, "int", toType)

		value, ok := customErr.GetMetadata(MetaKeyValue)
		assert.True(t, ok)
		assert.Equal(t, "abc", value)
	})

	t.Run("with complex value", func(t *testing.T) {
		complexValue := map[string]int{"a": 1, "b": 2}
		err := NewTypeConversionError("map[string]int", "[]int", complexValue)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		value, ok := customErr.GetMetadata(MetaKeyValue)
		assert.True(t, ok)
		assert.Contains(t, value, "map[")
	})

	t.Run("with nil value", func(t *testing.T) {
		err := NewTypeConversionError("nil", "string", nil)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		value, ok := customErr.GetMetadata(MetaKeyValue)
		assert.True(t, ok)
		assert.Equal(t, "<nil>", value)
	})
}

// TestNewTemplateNotFoundError tests template not found error creation
func TestNewTemplateNotFoundError(t *testing.T) {
	t.Run("basic template not found", func(t *testing.T) {
		err := NewTemplateNotFoundError("header")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgTemplateNotFound)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		name, ok := customErr.GetMetadata(MetaKeyTemplateName)
		assert.True(t, ok)
		assert.Equal(t, "header", name)
	})

	t.Run("with nested template name", func(t *testing.T) {
		err := NewTemplateNotFoundError("components.user.profile")

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		name, ok := customErr.GetMetadata(MetaKeyTemplateName)
		assert.True(t, ok)
		assert.Equal(t, "components.user.profile", name)
	})
}

// TestNewTemplateExistsError tests template already registered error creation
func TestNewTemplateExistsError(t *testing.T) {
	t.Run("basic template exists", func(t *testing.T) {
		err := NewTemplateExistsError("footer")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgTemplateAlreadyExists)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		name, ok := customErr.GetMetadata(MetaKeyTemplateName)
		assert.True(t, ok)
		assert.Equal(t, "footer", name)
	})
}

// TestNewTemplateDepthError tests template depth exceeded error creation
func TestNewTemplateDepthError(t *testing.T) {
	t.Run("basic depth exceeded", func(t *testing.T) {
		err := NewTemplateDepthError(11, 10)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgTemplateDepthExceeded)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		currentDepth, ok := customErr.GetMetadata(MetaKeyCurrentDepth)
		assert.True(t, ok)
		assert.Equal(t, "11", currentDepth)

		maxDepth, ok := customErr.GetMetadata(MetaKeyMaxDepth)
		assert.True(t, ok)
		assert.Equal(t, "10", maxDepth)
	})

	t.Run("with large depth values", func(t *testing.T) {
		err := NewTemplateDepthError(1000, 999)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		currentDepth, ok := customErr.GetMetadata(MetaKeyCurrentDepth)
		assert.True(t, ok)
		assert.Equal(t, "1000", currentDepth)
	})

	t.Run("with zero values", func(t *testing.T) {
		err := NewTemplateDepthError(0, 0)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		currentDepth, ok := customErr.GetMetadata(MetaKeyCurrentDepth)
		assert.True(t, ok)
		assert.Equal(t, "0", currentDepth)
	})
}

// TestNewReservedTemplateNameError tests reserved namespace error creation
func TestNewReservedTemplateNameError(t *testing.T) {
	t.Run("basic reserved name", func(t *testing.T) {
		err := NewReservedTemplateNameError("prompty.custom")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgReservedTemplateName)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		name, ok := customErr.GetMetadata(MetaKeyTemplateName)
		assert.True(t, ok)
		assert.Equal(t, "prompty.custom", name)
	})

	t.Run("with builtin tag name", func(t *testing.T) {
		err := NewReservedTemplateNameError(TagNameVar)

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		name, ok := customErr.GetMetadata(MetaKeyTemplateName)
		assert.True(t, ok)
		assert.Equal(t, TagNameVar, name)
	})
}

// TestNewEmptyTemplateNameError tests empty template name error creation
func TestNewEmptyTemplateNameError(t *testing.T) {
	t.Run("basic empty name", func(t *testing.T) {
		err := NewEmptyTemplateNameError()

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEmptyTemplateName)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))
	})
}

// TestNewEngineNotAvailableError tests engine not available error creation
func TestNewEngineNotAvailableError(t *testing.T) {
	t.Run("basic engine not available", func(t *testing.T) {
		err := NewEngineNotAvailableError()

		require.Error(t, err)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata contains tag information
		tag, ok := customErr.GetMetadata(MetaKeyTag)
		assert.True(t, ok)
		assert.Equal(t, TagNameInclude, tag)
	})
}

// TestNewFuncRegistrationError tests function registration error creation
func TestNewFuncRegistrationError(t *testing.T) {
	t.Run("with function name", func(t *testing.T) {
		err := NewFuncRegistrationError(ErrMsgFuncAlreadyExists, "upper")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncAlreadyExists)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// Verify metadata
		funcName, ok := customErr.GetMetadata(MetaKeyFuncName)
		assert.True(t, ok)
		assert.Equal(t, "upper", funcName)
	})

	t.Run("without function name", func(t *testing.T) {
		err := NewFuncRegistrationError(ErrMsgFuncEmptyName, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncEmptyName)

		var customErr *cuserr.CustomError
		require.True(t, errors.As(err, &customErr))

		// No function name should be set in metadata
		_, ok := customErr.GetMetadata(MetaKeyFuncName)
		assert.False(t, ok)
	})

	t.Run("with nil function error", func(t *testing.T) {
		err := NewFuncRegistrationError(ErrMsgFuncNilFunc, "customFunc")

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncNilFunc)
	})
}

// TestPosition tests Position type
func TestPosition(t *testing.T) {
	t.Run("string representation", func(t *testing.T) {
		pos := Position{Line: 5, Column: 10, Offset: 50}
		str := pos.String()

		assert.Equal(t, "line 5, column 10", str)
	})

	t.Run("zero position", func(t *testing.T) {
		pos := Position{Line: 0, Column: 0, Offset: 0}
		str := pos.String()

		assert.Equal(t, "line 0, column 0", str)
	})

	t.Run("large position values", func(t *testing.T) {
		pos := Position{Line: 9999, Column: 8888, Offset: 777777}
		str := pos.String()

		assert.Equal(t, "line 9999, column 8888", str)
	})
}

// TestErrorConstants verifies all error message constants are defined and non-empty
func TestErrorConstants(t *testing.T) {
	t.Run("all error message constants non-empty", func(t *testing.T) {
		// Parse errors
		assert.NotEmpty(t, ErrMsgParseFailed)
		assert.NotEmpty(t, ErrMsgInvalidSyntax)
		assert.NotEmpty(t, ErrMsgUnexpectedChar)
		assert.NotEmpty(t, ErrMsgUnterminatedTag)
		assert.NotEmpty(t, ErrMsgUnterminatedStr)
		assert.NotEmpty(t, ErrMsgInvalidEscape)
		assert.NotEmpty(t, ErrMsgUnexpectedEOF)
		assert.NotEmpty(t, ErrMsgMismatchedTag)
		assert.NotEmpty(t, ErrMsgInvalidTagName)
		assert.NotEmpty(t, ErrMsgEmptyTagName)
		assert.NotEmpty(t, ErrMsgNestedRawBlock)

		// Execution errors
		assert.NotEmpty(t, ErrMsgUnknownTag)
		assert.NotEmpty(t, ErrMsgUnknownResolver)
		assert.NotEmpty(t, ErrMsgResolverFailed)
		assert.NotEmpty(t, ErrMsgVariableNotFound)
		assert.NotEmpty(t, ErrMsgInvalidPath)
		assert.NotEmpty(t, ErrMsgEmptyPath)
		assert.NotEmpty(t, ErrMsgExecutionFailed)

		// Validation errors
		assert.NotEmpty(t, ErrMsgMissingAttribute)
		assert.NotEmpty(t, ErrMsgInvalidAttribute)

		// Registry errors
		assert.NotEmpty(t, ErrMsgResolverExists)

		// Type conversion errors
		assert.NotEmpty(t, ErrMsgTypeConversion)

		// Template errors
		assert.NotEmpty(t, ErrMsgTemplateNotFound)
		assert.NotEmpty(t, ErrMsgTemplateAlreadyExists)
		assert.NotEmpty(t, ErrMsgTemplateDepthExceeded)
		assert.NotEmpty(t, ErrMsgInvalidTemplateName)
		assert.NotEmpty(t, ErrMsgEmptyTemplateName)
		assert.NotEmpty(t, ErrMsgMissingTemplateAttr)
		assert.NotEmpty(t, ErrMsgEngineNotAvailable)
		assert.NotEmpty(t, ErrMsgReservedTemplateName)

		// Custom function errors
		assert.NotEmpty(t, ErrMsgFuncNilFunc)
		assert.NotEmpty(t, ErrMsgFuncEmptyName)
		assert.NotEmpty(t, ErrMsgFuncAlreadyExists)
	})

	t.Run("all error code constants non-empty", func(t *testing.T) {
		assert.NotEmpty(t, ErrCodeParse)
		assert.NotEmpty(t, ErrCodeExec)
		assert.NotEmpty(t, ErrCodeValidation)
		assert.NotEmpty(t, ErrCodeRegistry)
		assert.NotEmpty(t, ErrCodeTemplate)
		assert.NotEmpty(t, ErrCodeFunc)
	})
}

// TestErrorMetadataKeys verifies all metadata key constants are defined
func TestErrorMetadataKeys(t *testing.T) {
	t.Run("all metadata keys non-empty", func(t *testing.T) {
		assert.NotEmpty(t, MetaKeyLine)
		assert.NotEmpty(t, MetaKeyColumn)
		assert.NotEmpty(t, MetaKeyOffset)
		assert.NotEmpty(t, MetaKeyTag)
		assert.NotEmpty(t, MetaKeyResolver)
		assert.NotEmpty(t, MetaKeyVariable)
		assert.NotEmpty(t, MetaKeyAttribute)
		assert.NotEmpty(t, MetaKeyExpected)
		assert.NotEmpty(t, MetaKeyActual)
		assert.NotEmpty(t, MetaKeyPath)
		assert.NotEmpty(t, MetaKeyValue)
		assert.NotEmpty(t, MetaKeyTemplateName)
		assert.NotEmpty(t, MetaKeyCurrentDepth)
		assert.NotEmpty(t, MetaKeyMaxDepth)
		assert.NotEmpty(t, MetaKeyFuncName)
		assert.NotEmpty(t, MetaKeyReason)
		assert.NotEmpty(t, MetaKeyFromType)
		assert.NotEmpty(t, MetaKeyToType)
	})
}
