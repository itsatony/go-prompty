package prompty

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateTokens_Empty(t *testing.T) {
	result := EstimateTokens("")

	assert.Equal(t, 0, result.Characters)
	assert.Equal(t, 0, result.Words)
	assert.Equal(t, 0, result.Lines)
	assert.Equal(t, 0, result.EstimatedGPT)
	assert.Equal(t, 0, result.EstimatedClaude)
	assert.Equal(t, 0, result.EstimatedGeneric)
}

func TestEstimateTokens_SimpleText(t *testing.T) {
	text := "Hello, world!"
	result := EstimateTokens(text)

	assert.Equal(t, 13, result.Characters)
	assert.Equal(t, 2, result.Words)
	assert.Equal(t, 1, result.Lines)
	assert.True(t, result.EstimatedGPT > 0)
	assert.True(t, result.EstimatedClaude > 0)
	assert.True(t, result.EstimatedGeneric > 0)
}

func TestEstimateTokens_MultipleLines(t *testing.T) {
	text := "Line 1\nLine 2\nLine 3"
	result := EstimateTokens(text)

	assert.Equal(t, 3, result.Lines)
	assert.Equal(t, 6, result.Words)
}

func TestEstimateTokens_LongText(t *testing.T) {
	// Generate a 1000-character text
	text := strings.Repeat("Hello world. ", 100)
	result := EstimateTokens(text)

	// ~4 chars per token for GPT
	expectedGPTMin := len(text) / 5 // At least 200 tokens
	expectedGPTMax := len(text) / 3 // At most 400 tokens

	assert.True(t, result.EstimatedGPT >= expectedGPTMin, "GPT estimate too low")
	assert.True(t, result.EstimatedGPT <= expectedGPTMax, "GPT estimate too high")
}

func TestEstimateTokens_Unicode(t *testing.T) {
	text := "Hello 世界! Привет мир!"
	result := EstimateTokens(text)

	assert.True(t, result.Characters > 0)
	assert.True(t, result.NonASCIIRatio > 0)
	// Non-ASCII content should have different estimation
	assert.True(t, result.EstimatedGeneric > 0)
}

func TestEstimateTokens_HighNonASCII(t *testing.T) {
	// Chinese text has higher token density
	text := "这是一个测试文本，用于测试中文文本的令牌估计"
	result := EstimateTokens(text)

	assert.True(t, result.NonASCIIRatio > 0.3)
	// Higher non-ASCII ratio should result in more conservative token estimate
	assert.True(t, result.EstimatedGeneric > result.Characters/4)
}

func TestEstimateTokens_Whitespace(t *testing.T) {
	text := "    lots    of    spaces    " // More whitespace
	result := EstimateTokens(text)

	assert.True(t, result.WhitespaceRatio > 0.5)
}

func TestEstimateTokens_CodeContent(t *testing.T) {
	text := `func main() {
    fmt.Println("Hello, World!")
}`
	result := EstimateTokens(text)

	assert.True(t, result.Characters > 0)
	assert.True(t, result.Lines == 3)
	assert.True(t, result.EstimatedGPT > 0)
}

func TestTemplate_EstimateTokens(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" /~}! Welcome to {~prompty.var name="app" /~}.`)
	require.NoError(t, err)

	estimate, err := tmpl.EstimateTokens(context.Background(), map[string]any{
		"user": "Alice",
		"app":  "MyApp",
	})
	require.NoError(t, err)

	assert.True(t, estimate.Characters > 0)
	assert.Contains(t, []int{estimate.EstimatedGPT, estimate.EstimatedClaude, estimate.EstimatedGeneric}, estimate.EstimatedGPT)
}

func TestTemplate_EstimateTokensDryRun(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" /~}!`)
	require.NoError(t, err)

	// Dry run with missing data
	estimate := tmpl.EstimateTokensDryRun(context.Background(), map[string]any{})

	// Should include placeholder in estimate
	assert.True(t, estimate.Characters > 0)
	assert.True(t, estimate.EstimatedGeneric > 0)
}

func TestTemplate_EstimateSourceTokens(t *testing.T) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}! Your items: {~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	estimate := tmpl.EstimateSourceTokens()

	assert.Equal(t, len(source), estimate.Characters)
	assert.True(t, estimate.EstimatedGeneric > 0)
}

func TestTokenEstimate_EstimateCost(t *testing.T) {
	estimate := &TokenEstimate{
		Characters:       1000,
		EstimatedGPT:     250,
		EstimatedClaude:  250,
		EstimatedLlama:   286,
		EstimatedGeneric: 333,
	}

	cost := estimate.EstimateCost()

	assert.Equal(t, 333, cost.InputTokens)
	assert.True(t, cost.GPT4Cost > 0)
	assert.True(t, cost.GPT4oCost > 0)
	assert.True(t, cost.GPT35Cost > 0)
	assert.True(t, cost.ClaudeOpusCost > 0)
	assert.True(t, cost.ClaudeSonnetCost > 0)
	assert.True(t, cost.ClaudeHaikuCost > 0)

	// GPT-4 should be more expensive than GPT-4o
	assert.True(t, cost.GPT4Cost > cost.GPT4oCost)
	// Claude Opus should be more expensive than Haiku
	assert.True(t, cost.ClaudeOpusCost > cost.ClaudeHaikuCost)
}

func TestTokenEstimate_EstimateCostForModel(t *testing.T) {
	estimate := &TokenEstimate{
		EstimatedGPT:     100,
		EstimatedClaude:  100,
		EstimatedLlama:   115,
		EstimatedGeneric: 133,
	}

	tests := []struct {
		model    string
		price    float64
		expected float64
	}{
		{"gpt-4", 0.03, 0.003},
		{"gpt-4o", 0.005, 0.0005},
		{"claude-3-opus", 0.015, 0.0015},
		{"llama-3", 0.001, 0.000115},
		{"unknown", 0.01, 0.00133},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			cost := estimate.EstimateCostForModel(tt.model, tt.price)
			assert.InDelta(t, tt.expected, cost, 0.0001)
		})
	}
}

func TestNewTokenBudget(t *testing.T) {
	budget := NewTokenBudget(8000, 2000)

	assert.Equal(t, 8000, budget.MaxTokens)
	assert.Equal(t, 2000, budget.ReservedForResponse)
	assert.Equal(t, 6000, budget.AvailableForPrompt)
}

func TestNewTokenBudget_ReservedExceedsMax(t *testing.T) {
	budget := NewTokenBudget(1000, 2000)

	assert.Equal(t, 1000, budget.MaxTokens)
	assert.Equal(t, 2000, budget.ReservedForResponse)
	assert.Equal(t, 0, budget.AvailableForPrompt)
}

func TestTokenBudget_FitsWithin(t *testing.T) {
	budget := NewTokenBudget(8000, 2000) // 6000 available

	smallEstimate := &TokenEstimate{EstimatedGeneric: 1000}
	largeEstimate := &TokenEstimate{EstimatedGeneric: 7000}

	assert.True(t, budget.FitsWithin(smallEstimate))
	assert.False(t, budget.FitsWithin(largeEstimate))
}

func TestTokenBudget_RemainingTokens(t *testing.T) {
	budget := NewTokenBudget(8000, 2000) // 6000 available

	estimate := &TokenEstimate{EstimatedGeneric: 1000}

	remaining := budget.RemainingTokens(estimate)
	assert.Equal(t, 5000, remaining)
}

func TestTokenBudget_RemainingTokens_NegativeClampedToZero(t *testing.T) {
	budget := NewTokenBudget(8000, 2000) // 6000 available

	estimate := &TokenEstimate{EstimatedGeneric: 7000}

	remaining := budget.RemainingTokens(estimate)
	assert.Equal(t, 0, remaining)
}

func TestTokenBudget_OverageTokens(t *testing.T) {
	budget := NewTokenBudget(8000, 2000) // 6000 available

	smallEstimate := &TokenEstimate{EstimatedGeneric: 1000}
	largeEstimate := &TokenEstimate{EstimatedGeneric: 7000}

	assert.Equal(t, 0, budget.OverageTokens(smallEstimate))
	assert.Equal(t, 1000, budget.OverageTokens(largeEstimate))
}

func TestPresetBudgets(t *testing.T) {
	gptBudget := NewGPT4TurboBudget(4000)
	assert.Equal(t, ContextGPT4Turbo, gptBudget.MaxTokens)
	assert.Equal(t, ContextGPT4Turbo-4000, gptBudget.AvailableForPrompt)

	claudeBudget := NewClaudeBudget(8000)
	assert.Equal(t, ContextClaudeOpus, claudeBudget.MaxTokens)
	assert.Equal(t, ContextClaudeOpus-8000, claudeBudget.AvailableForPrompt)
}

func TestContextWindowConstants(t *testing.T) {
	assert.Equal(t, 128000, ContextGPT4Turbo)
	assert.Equal(t, 8192, ContextGPT4)
	assert.Equal(t, 16385, ContextGPT35)
	assert.Equal(t, 200000, ContextClaudeOpus)
	assert.Equal(t, 200000, ContextClaudeSonnet)
	assert.Equal(t, 200000, ContextClaudeHaiku)
	assert.Equal(t, 8192, ContextLlama3)
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"one two three four five", 5},
		{"no-hyphen-words", 1},
		{"line1\nline2", 2},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			assert.Equal(t, tt.expected, countWords(tt.text))
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"single line", 1},
		{"line1\nline2", 2},
		{"line1\nline2\nline3", 3},
		{"\n\n\n", 4},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			assert.Equal(t, tt.expected, countLines(tt.text))
		})
	}
}

func TestCountWhitespace(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"nospace", 0},
		{"one space", 1},
		{"  two  ", 4},
		{"tab\there", 1},
		{"newline\nhere", 1},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			assert.Equal(t, tt.expected, countWhitespace(tt.text))
		})
	}
}

func TestCountNonASCII(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"hello", 0},
		{"hello 世界", 2},
		{"Привет", 6},
		{"emoji 🎉", 1}, // emoji is one rune
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			assert.Equal(t, tt.expected, countNonASCII(tt.text))
		})
	}
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		chars    int
		divisor  float64
		expected int
	}{
		{0, 4.0, 0},
		{100, 4.0, 25},
		{100, 3.0, 33}, // rounds to 33.33... -> 33
		{1000, 4.0, 250},
		{99, 4.0, 25}, // 24.75 rounds to 25
	}

	for _, tt := range tests {
		result := estimateTokenCount(tt.chars, tt.divisor)
		assert.Equal(t, tt.expected, result)
	}
}

func TestEstimateTokens_RealWorldPrompt(t *testing.T) {
	prompt := `You are a helpful assistant. Please analyze the following code and provide suggestions for improvement:

` + "```go\n" + `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
` + "```\n" + `
Focus on:
1. Code organization
2. Error handling
3. Performance
4. Best practices`

	estimate := EstimateTokens(prompt)

	// Sanity checks
	assert.True(t, estimate.Characters > 200)
	assert.True(t, estimate.Words > 30)
	assert.True(t, estimate.Lines > 10)
	assert.True(t, estimate.EstimatedGPT > 50)
	assert.True(t, estimate.EstimatedGPT < 200)
}

func BenchmarkEstimateTokens(b *testing.B) {
	text := strings.Repeat("Hello, this is a test sentence. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(text)
	}
}

func BenchmarkEstimateTokens_LongText(b *testing.B) {
	text := strings.Repeat("Hello, this is a test sentence with some variation. ", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(text)
	}
}
