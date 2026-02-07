package prompty

import (
	"context"
	"testing"

	"github.com/itsatony/go-prompty/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TemplateBody tests
// ---------------------------------------------------------------------------

func TestTemplateBody_WithYAMLFrontmatter(t *testing.T) {
	source := "---\nname: test-prompt\ndescription: a test prompt\n---\n{~prompty.message role=\"system\"~}Hello{~/prompty.message~}"

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	body := tmpl.TemplateBody()
	// The body should NOT include the YAML frontmatter block
	assert.NotContains(t, body, "---")
	assert.NotContains(t, body, "name: test-prompt")
	assert.NotContains(t, body, "description: a test prompt")
	// The body should include the template content
	assert.Contains(t, body, "prompty.message")
}

func TestTemplateBody_WithoutFrontmatter(t *testing.T) {
	source := `{~prompty.message role="system"~}You are a helpful assistant.{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	body := tmpl.TemplateBody()
	// Without frontmatter, the body should match the source
	assert.Equal(t, source, body)
}

func TestTemplateBody_PlainText(t *testing.T) {
	source := "Hello world, no tags here."

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	body := tmpl.TemplateBody()
	assert.Equal(t, source, body)
}

func TestTemplateBody_FrontmatterOnlyFields(t *testing.T) {
	source := "---\nname: minimal\ndescription: minimal prompt\n---\nBody content only"

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	body := tmpl.TemplateBody()
	assert.NotContains(t, body, "name: minimal")
	assert.Contains(t, body, "Body content only")
}

// ---------------------------------------------------------------------------
// ExecuteAndExtractMessages tests
// ---------------------------------------------------------------------------

func TestExecuteAndExtractMessages_SingleSystemMessage(t *testing.T) {
	source := `{~prompty.message role="system"~}You are a helpful assistant.{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", messages[0].Content)
	assert.False(t, messages[0].Cache)
}

func TestExecuteAndExtractMessages_MultipleMessages(t *testing.T) {
	source := `{~prompty.message role="system"~}You are a helpful assistant.{~/prompty.message~}
{~prompty.message role="user"~}Hello there!{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", messages[0].Content)

	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "Hello there!", messages[1].Content)
}

func TestExecuteAndExtractMessages_WithVariables(t *testing.T) {
	source := `{~prompty.message role="user"~}{~prompty.var name="query" /~}{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]any{
		"query": "What is the meaning of life?",
	}
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, data)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "What is the meaning of life?", messages[0].Content)
	assert.False(t, messages[0].Cache)
}

func TestExecuteAndExtractMessages_WithCacheTrue(t *testing.T) {
	source := `{~prompty.message role="system" cache="true"~}Cached system message.{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "Cached system message.", messages[0].Content)
	assert.True(t, messages[0].Cache)
}

func TestExecuteAndExtractMessages_NoMessages(t *testing.T) {
	source := "Just plain text with no message tags."

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, messages)
}

func TestExecuteAndExtractMessages_AllRoles(t *testing.T) {
	source := `{~prompty.message role="system"~}System prompt.{~/prompty.message~}
{~prompty.message role="user"~}User input.{~/prompty.message~}
{~prompty.message role="assistant"~}Assistant response.{~/prompty.message~}
{~prompty.message role="tool"~}Tool result.{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 4)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "System prompt.", messages[0].Content)

	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "User input.", messages[1].Content)

	assert.Equal(t, RoleAssistant, messages[2].Role)
	assert.Equal(t, "Assistant response.", messages[2].Content)

	assert.Equal(t, RoleTool, messages[3].Role)
	assert.Equal(t, "Tool result.", messages[3].Content)
}

func TestExecuteAndExtractMessages_WithFrontmatter(t *testing.T) {
	source := "---\nname: chat-prompt\ndescription: a chat prompt\n---\n{~prompty.message role=\"system\"~}Hello from frontmatter template.{~/prompty.message~}"

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "Hello from frontmatter template.", messages[0].Content)
}

func TestExecuteAndExtractMessages_MultipleVariablesAndMessages(t *testing.T) {
	source := `{~prompty.message role="system"~}You are {~prompty.var name="persona" /~}.{~/prompty.message~}
{~prompty.message role="user"~}{~prompty.var name="query" /~}{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]any{
		"persona": "a research assistant",
		"query":   "Find me papers on quantum computing.",
	}
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, data)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a research assistant.", messages[0].Content)

	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "Find me papers on quantum computing.", messages[1].Content)
}

func TestExecuteAndExtractMessages_MixedCacheFlags(t *testing.T) {
	source := `{~prompty.message role="system" cache="true"~}Cached system prompt.{~/prompty.message~}
{~prompty.message role="user"~}Not cached user input.{~/prompty.message~}
{~prompty.message role="assistant" cache="true"~}Cached assistant response.{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 3)

	assert.True(t, messages[0].Cache)
	assert.False(t, messages[1].Cache)
	assert.True(t, messages[2].Cache)
}

func TestExecuteAndExtractMessages_MultilineContent(t *testing.T) {
	source := `{~prompty.message role="system"~}
You are a helpful assistant.
You always respond politely.
You cite your sources.
{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleSystem, messages[0].Role)
	// Content should be trimmed but preserve internal newlines
	assert.Contains(t, messages[0].Content, "You are a helpful assistant.")
	assert.Contains(t, messages[0].Content, "You always respond politely.")
	assert.Contains(t, messages[0].Content, "You cite your sources.")
}

// ---------------------------------------------------------------------------
// ExtractMessagesFromOutput tests
// ---------------------------------------------------------------------------

func TestExtractMessagesFromOutput_EmptyString(t *testing.T) {
	messages := ExtractMessagesFromOutput("")
	assert.Nil(t, messages)
}

func TestExtractMessagesFromOutput_NoMarkers(t *testing.T) {
	messages := ExtractMessagesFromOutput("Just plain text with no markers at all.")
	assert.Nil(t, messages)
}

func TestExtractMessagesFromOutput_ValidSingleMessage(t *testing.T) {
	output := internal.MessageStartMarker + "system:false:You are a helpful assistant." + internal.MessageEndMarker

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", messages[0].Content)
	assert.False(t, messages[0].Cache)
}

func TestExtractMessagesFromOutput_ValidMultipleMessages(t *testing.T) {
	output := internal.MessageStartMarker + "system:true:System prompt." + internal.MessageEndMarker +
		internal.MessageStartMarker + "user:false:User question." + internal.MessageEndMarker +
		internal.MessageStartMarker + "assistant:false:Assistant answer." + internal.MessageEndMarker

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 3)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "System prompt.", messages[0].Content)
	assert.True(t, messages[0].Cache)

	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "User question.", messages[1].Content)
	assert.False(t, messages[1].Cache)

	assert.Equal(t, RoleAssistant, messages[2].Role)
	assert.Equal(t, "Assistant answer.", messages[2].Content)
	assert.False(t, messages[2].Cache)
}

func TestExtractMessagesFromOutput_WithSurroundingText(t *testing.T) {
	output := "Some prefix text\n" +
		internal.MessageStartMarker + "user:false:Hello!" + internal.MessageEndMarker +
		"\nSome suffix text"

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "Hello!", messages[0].Content)
}

func TestExtractMessagesFromOutput_ContentWithColons(t *testing.T) {
	output := internal.MessageStartMarker + "user:false:Time is 12:30:00 PM" + internal.MessageEndMarker

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "Time is 12:30:00 PM", messages[0].Content)
}

func TestExtractMessagesFromOutput_CacheTrue(t *testing.T) {
	output := internal.MessageStartMarker + "system:true:Cached content." + internal.MessageEndMarker

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "Cached content.", messages[0].Content)
	assert.True(t, messages[0].Cache)
}

func TestExtractMessagesFromOutput_WhitespaceContent(t *testing.T) {
	output := internal.MessageStartMarker + "user:false:   \n\t  " + internal.MessageEndMarker

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "", messages[0].Content) // Whitespace trimmed to empty
}

func TestExtractMessagesFromOutput_EmptyContent(t *testing.T) {
	output := internal.MessageStartMarker + "user:false:" + internal.MessageEndMarker

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "", messages[0].Content)
}

func TestExtractMessagesFromOutput_ReturnsNilNotEmpty(t *testing.T) {
	// Verify the return type is nil (not an empty slice) when no messages found.
	// This matches the contract: internal.ExtractMessages returns nil for no messages.
	messages := ExtractMessagesFromOutput("no messages here")
	assert.Nil(t, messages)

	// Also test with empty string
	messages = ExtractMessagesFromOutput("")
	assert.Nil(t, messages)
}

func TestExtractMessagesFromOutput_UnicodeContent(t *testing.T) {
	content := "Hello from Unicode land: cafe au lait, nino, resume"
	output := internal.MessageStartMarker + "user:false:" + content + internal.MessageEndMarker

	messages := ExtractMessagesFromOutput(output)
	require.Len(t, messages, 1)

	assert.Equal(t, content, messages[0].Content)
}

// ---------------------------------------------------------------------------
// Integration: round-trip through engine execution
// ---------------------------------------------------------------------------

func TestExecuteAndExtractMessages_RoundTrip(t *testing.T) {
	// This test verifies the full round trip: parse -> execute -> extract
	// using the engine's Execute method and then extracting messages from raw output.
	source := `{~prompty.message role="system"~}You are helpful.{~/prompty.message~}
{~prompty.message role="user"~}Hi{~/prompty.message~}`

	engine := MustNew()
	ctx := context.Background()

	// First, use Execute to get raw output
	rawOutput, err := engine.Execute(ctx, source, nil)
	require.NoError(t, err)

	// Then extract messages from raw output
	messagesFromRaw := ExtractMessagesFromOutput(rawOutput)
	require.Len(t, messagesFromRaw, 2)

	// Also use the template's convenience method
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	messagesFromTmpl, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messagesFromTmpl, 2)

	// Both should produce identical results
	assert.Equal(t, messagesFromRaw[0].Role, messagesFromTmpl[0].Role)
	assert.Equal(t, messagesFromRaw[0].Content, messagesFromTmpl[0].Content)
	assert.Equal(t, messagesFromRaw[0].Cache, messagesFromTmpl[0].Cache)

	assert.Equal(t, messagesFromRaw[1].Role, messagesFromTmpl[1].Role)
	assert.Equal(t, messagesFromRaw[1].Content, messagesFromTmpl[1].Content)
	assert.Equal(t, messagesFromRaw[1].Cache, messagesFromTmpl[1].Cache)
}

func TestExecuteAndExtractMessages_ConditionalMessage(t *testing.T) {
	source := `{~prompty.message role="system"~}System prompt.{~/prompty.message~}
{~prompty.if eval="showUser"~}{~prompty.message role="user"~}User message.{~/prompty.message~}{~/prompty.if~}`

	engine := MustNew()
	ctx := context.Background()

	// With condition true: both messages
	messages, err := engine.Parse(source)
	require.NoError(t, err)

	result, err := messages.ExecuteAndExtractMessages(ctx, map[string]any{
		"showUser": true,
	})
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, RoleSystem, result[0].Role)
	assert.Equal(t, RoleUser, result[1].Role)

	// With condition false: only system message
	result, err = messages.ExecuteAndExtractMessages(ctx, map[string]any{
		"showUser": false,
	})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, RoleSystem, result[0].Role)
}

func TestExecuteAndExtractMessages_CacheFalseExplicit(t *testing.T) {
	source := `{~prompty.message role="system" cache="false"~}Not cached.{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	assert.False(t, messages[0].Cache)
}

func TestExecuteAndExtractMessages_ReusableTemplate(t *testing.T) {
	// Verify that the same parsed template can be executed multiple times
	// with different data and always produces correct messages.
	source := `{~prompty.message role="user"~}{~prompty.var name="query" /~}{~/prompty.message~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	ctx := context.Background()

	msgs1, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{"query": "first"})
	require.NoError(t, err)
	require.Len(t, msgs1, 1)
	assert.Equal(t, "first", msgs1[0].Content)

	msgs2, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{"query": "second"})
	require.NoError(t, err)
	require.Len(t, msgs2, 1)
	assert.Equal(t, "second", msgs2[0].Content)
}
