package prompty

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// ExecutionConfig represents the v2.0 namespaced execution configuration.
// It contains all parameters needed for LLM inference.
type ExecutionConfig struct {
	// Provider identifier (e.g., "openai", "anthropic", "gemini", "vllm")
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	// Model identifier (e.g., "gpt-4", "claude-sonnet-4-5")
	Model string `yaml:"model,omitempty" json:"model,omitempty"`

	// Common inference parameters
	Temperature   *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	MaxTokens     *int     `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	TopP          *float64 `yaml:"top_p,omitempty" json:"top_p,omitempty"`
	TopK          *int     `yaml:"top_k,omitempty" json:"top_k,omitempty"`
	StopSequences []string `yaml:"stop_sequences,omitempty" json:"stop_sequences,omitempty"`

	// Extended inference parameters (v2.3)
	MinP              *float64           `yaml:"min_p,omitempty" json:"min_p,omitempty"`
	RepetitionPenalty *float64           `yaml:"repetition_penalty,omitempty" json:"repetition_penalty,omitempty"`
	Seed              *int               `yaml:"seed,omitempty" json:"seed,omitempty"`
	Logprobs          *int               `yaml:"logprobs,omitempty" json:"logprobs,omitempty"`
	StopTokenIDs      []int              `yaml:"stop_token_ids,omitempty" json:"stop_token_ids,omitempty"`
	LogitBias         map[string]float64 `yaml:"logit_bias,omitempty" json:"logit_bias,omitempty"`

	// Extended thinking configuration (Anthropic)
	Thinking *ThinkingConfig `yaml:"thinking,omitempty" json:"thinking,omitempty"`

	// Structured output configuration
	ResponseFormat *ResponseFormat `yaml:"response_format,omitempty" json:"response_format,omitempty"`
	GuidedDecoding *GuidedDecoding `yaml:"guided_decoding,omitempty" json:"guided_decoding,omitempty"`

	// v2.5 Modality — execution intent signal (e.g., "text", "image", "audio_speech", "embedding")
	Modality string `yaml:"modality,omitempty" json:"modality,omitempty"`

	// v2.5 Media generation configs
	Image     *ImageConfig     `yaml:"image,omitempty" json:"image,omitempty"`
	Audio     *AudioConfig     `yaml:"audio,omitempty" json:"audio,omitempty"`
	Embedding *EmbeddingConfig `yaml:"embedding,omitempty" json:"embedding,omitempty"`

	// v2.5 Execution mode configs
	Streaming *StreamingConfig `yaml:"streaming,omitempty" json:"streaming,omitempty"`
	Async     *AsyncConfig     `yaml:"async,omitempty" json:"async,omitempty"`

	// Provider-specific options (passthrough)
	ProviderOptions map[string]any `yaml:"provider_options,omitempty" json:"provider_options,omitempty"`
}

// ThinkingConfig configures extended thinking mode (Anthropic Claude).
type ThinkingConfig struct {
	Enabled      bool `yaml:"enabled" json:"enabled"`
	BudgetTokens *int `yaml:"budget_tokens,omitempty" json:"budget_tokens,omitempty"`
}

// Validate checks the execution config for consistency.
func (e *ExecutionConfig) Validate() error {
	if e == nil {
		return nil
	}

	// Validate temperature range if set
	if e.Temperature != nil {
		if *e.Temperature < 0.0 || *e.Temperature > 2.0 {
			return NewPromptValidationError(ErrMsgTemperatureOutOfRange, "")
		}
	}

	// Validate top_p range if set
	if e.TopP != nil {
		if *e.TopP < 0.0 || *e.TopP > 1.0 {
			return NewPromptValidationError(ErrMsgTopPOutOfRange, "")
		}
	}

	// Validate max_tokens if set
	if e.MaxTokens != nil && *e.MaxTokens <= 0 {
		return NewPromptValidationError(ErrMsgMaxTokensInvalid, "")
	}

	// Validate top_k if set
	if e.TopK != nil && *e.TopK < 0 {
		return NewPromptValidationError(ErrMsgTopKInvalid, "")
	}

	// Validate min_p range if set
	if e.MinP != nil {
		if *e.MinP < 0.0 || *e.MinP > 1.0 {
			return NewPromptValidationError(ErrMsgMinPOutOfRange, "")
		}
	}

	// Validate repetition_penalty if set
	if e.RepetitionPenalty != nil {
		if *e.RepetitionPenalty <= 0.0 {
			return NewPromptValidationError(ErrMsgRepetitionPenaltyOutOfRange, "")
		}
	}

	// Validate logprobs range if set
	if e.Logprobs != nil {
		if *e.Logprobs < 0 || *e.Logprobs > 20 {
			return NewPromptValidationError(ErrMsgLogprobsOutOfRange, "")
		}
	}

	// Validate stop_token_ids if set
	for _, id := range e.StopTokenIDs {
		if id < 0 {
			return NewPromptValidationError(ErrMsgStopTokenIDNegative, "")
		}
	}

	// Validate logit_bias values if set
	for _, v := range e.LogitBias {
		if v < -100.0 || v > 100.0 {
			return NewPromptValidationError(ErrMsgLogitBiasValueOutOfRange, "")
		}
	}

	// Validate thinking config if set
	if e.Thinking != nil && e.Thinking.Enabled {
		if e.Thinking.BudgetTokens != nil && *e.Thinking.BudgetTokens <= 0 {
			return NewPromptValidationError(ErrMsgThinkingBudgetInvalid, "")
		}
	}

	// Validate modality if set
	if e.Modality != "" && !isValidModality(e.Modality) {
		return NewPromptValidationError(ErrMsgInvalidModality, "")
	}

	// Validate media configs
	if e.Image != nil {
		if err := e.Image.Validate(); err != nil {
			return err
		}
	}
	if e.Audio != nil {
		if err := e.Audio.Validate(); err != nil {
			return err
		}
	}
	if e.Embedding != nil {
		if err := e.Embedding.Validate(); err != nil {
			return err
		}
	}
	if e.Streaming != nil {
		if err := e.Streaming.Validate(); err != nil {
			return err
		}
	}
	if e.Async != nil {
		if err := e.Async.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Clone creates a deep copy of the execution config.
func (e *ExecutionConfig) Clone() *ExecutionConfig {
	if e == nil {
		return nil
	}

	clone := &ExecutionConfig{
		Provider: e.Provider,
		Model:    e.Model,
	}

	if e.Temperature != nil {
		t := *e.Temperature
		clone.Temperature = &t
	}
	if e.MaxTokens != nil {
		m := *e.MaxTokens
		clone.MaxTokens = &m
	}
	if e.TopP != nil {
		tp := *e.TopP
		clone.TopP = &tp
	}
	if e.TopK != nil {
		tk := *e.TopK
		clone.TopK = &tk
	}
	if e.StopSequences != nil {
		clone.StopSequences = make([]string, len(e.StopSequences))
		copy(clone.StopSequences, e.StopSequences)
	}

	if e.MinP != nil {
		v := *e.MinP
		clone.MinP = &v
	}
	if e.RepetitionPenalty != nil {
		v := *e.RepetitionPenalty
		clone.RepetitionPenalty = &v
	}
	if e.Seed != nil {
		v := *e.Seed
		clone.Seed = &v
	}
	if e.Logprobs != nil {
		v := *e.Logprobs
		clone.Logprobs = &v
	}
	if e.StopTokenIDs != nil {
		clone.StopTokenIDs = make([]int, len(e.StopTokenIDs))
		copy(clone.StopTokenIDs, e.StopTokenIDs)
	}
	if e.LogitBias != nil {
		clone.LogitBias = make(map[string]float64, len(e.LogitBias))
		for k, v := range e.LogitBias {
			clone.LogitBias[k] = v
		}
	}

	if e.Thinking != nil {
		clone.Thinking = &ThinkingConfig{
			Enabled: e.Thinking.Enabled,
		}
		if e.Thinking.BudgetTokens != nil {
			bt := *e.Thinking.BudgetTokens
			clone.Thinking.BudgetTokens = &bt
		}
	}

	if e.ResponseFormat != nil {
		clone.ResponseFormat = cloneResponseFormat(e.ResponseFormat)
	}
	if e.GuidedDecoding != nil {
		clone.GuidedDecoding = cloneGuidedDecoding(e.GuidedDecoding)
	}

	// v2.5 media fields
	clone.Modality = e.Modality
	if e.Image != nil {
		clone.Image = e.Image.Clone()
	}
	if e.Audio != nil {
		clone.Audio = e.Audio.Clone()
	}
	if e.Embedding != nil {
		clone.Embedding = e.Embedding.Clone()
	}
	if e.Streaming != nil {
		clone.Streaming = e.Streaming.Clone()
	}
	if e.Async != nil {
		clone.Async = e.Async.Clone()
	}

	if e.ProviderOptions != nil {
		clone.ProviderOptions = make(map[string]any, len(e.ProviderOptions))
		for k, v := range e.ProviderOptions {
			clone.ProviderOptions[k] = v
		}
	}

	return clone
}

// cloneResponseFormat creates a deep copy of ResponseFormat.
func cloneResponseFormat(rf *ResponseFormat) *ResponseFormat {
	if rf == nil {
		return nil
	}
	clone := &ResponseFormat{
		Type: rf.Type,
	}
	if rf.JSONSchema != nil {
		clone.JSONSchema = &JSONSchemaSpec{
			Name:        rf.JSONSchema.Name,
			Description: rf.JSONSchema.Description,
			Strict:      rf.JSONSchema.Strict,
		}
		if rf.JSONSchema.Schema != nil {
			clone.JSONSchema.Schema = copySchema(rf.JSONSchema.Schema)
		}
		if rf.JSONSchema.AdditionalProperties != nil {
			ap := *rf.JSONSchema.AdditionalProperties
			clone.JSONSchema.AdditionalProperties = &ap
		}
		if rf.JSONSchema.PropertyOrdering != nil {
			clone.JSONSchema.PropertyOrdering = make([]string, len(rf.JSONSchema.PropertyOrdering))
			copy(clone.JSONSchema.PropertyOrdering, rf.JSONSchema.PropertyOrdering)
		}
	}
	if rf.Enum != nil {
		clone.Enum = &EnumConstraint{
			Description: rf.Enum.Description,
		}
		if rf.Enum.Values != nil {
			clone.Enum.Values = make([]string, len(rf.Enum.Values))
			copy(clone.Enum.Values, rf.Enum.Values)
		}
	}
	return clone
}

// cloneGuidedDecoding creates a deep copy of GuidedDecoding.
func cloneGuidedDecoding(gd *GuidedDecoding) *GuidedDecoding {
	if gd == nil {
		return nil
	}
	clone := &GuidedDecoding{
		Backend:           gd.Backend,
		Regex:             gd.Regex,
		Grammar:           gd.Grammar,
		WhitespacePattern: gd.WhitespacePattern,
	}
	if gd.JSON != nil {
		clone.JSON = copySchema(gd.JSON)
	}
	if gd.Choice != nil {
		clone.Choice = make([]string, len(gd.Choice))
		copy(clone.Choice, gd.Choice)
	}
	return clone
}

// GetProvider returns the provider or empty string.
func (e *ExecutionConfig) GetProvider() string {
	if e == nil {
		return ""
	}
	return e.Provider
}

// GetModel returns the model name or empty string.
func (e *ExecutionConfig) GetModel() string {
	if e == nil {
		return ""
	}
	return e.Model
}

// GetTemperature returns the temperature and whether it was set.
func (e *ExecutionConfig) GetTemperature() (float64, bool) {
	if e == nil || e.Temperature == nil {
		return 0, false
	}
	return *e.Temperature, true
}

// GetMaxTokens returns max_tokens and whether it was set.
func (e *ExecutionConfig) GetMaxTokens() (int, bool) {
	if e == nil || e.MaxTokens == nil {
		return 0, false
	}
	return *e.MaxTokens, true
}

// GetTopP returns top_p and whether it was set.
func (e *ExecutionConfig) GetTopP() (float64, bool) {
	if e == nil || e.TopP == nil {
		return 0, false
	}
	return *e.TopP, true
}

// GetTopK returns top_k and whether it was set.
func (e *ExecutionConfig) GetTopK() (int, bool) {
	if e == nil || e.TopK == nil {
		return 0, false
	}
	return *e.TopK, true
}

// GetStopSequences returns stop sequences or nil.
func (e *ExecutionConfig) GetStopSequences() []string {
	if e == nil {
		return nil
	}
	return e.StopSequences
}

// GetThinking returns the thinking config or nil.
func (e *ExecutionConfig) GetThinking() *ThinkingConfig {
	if e == nil {
		return nil
	}
	return e.Thinking
}

// GetResponseFormat returns the response format or nil.
func (e *ExecutionConfig) GetResponseFormat() *ResponseFormat {
	if e == nil {
		return nil
	}
	return e.ResponseFormat
}

// GetGuidedDecoding returns the guided decoding config or nil.
func (e *ExecutionConfig) GetGuidedDecoding() *GuidedDecoding {
	if e == nil {
		return nil
	}
	return e.GuidedDecoding
}

// HasThinking returns true if thinking is configured and enabled.
func (e *ExecutionConfig) HasThinking() bool {
	return e != nil && e.Thinking != nil && e.Thinking.Enabled
}

// HasResponseFormat returns true if response format is configured.
func (e *ExecutionConfig) HasResponseFormat() bool {
	return e != nil && e.ResponseFormat != nil
}

// HasGuidedDecoding returns true if guided decoding is configured.
func (e *ExecutionConfig) HasGuidedDecoding() bool {
	return e != nil && e.GuidedDecoding != nil
}

// GetMinP returns min_p and whether it was set.
func (e *ExecutionConfig) GetMinP() (float64, bool) {
	if e == nil || e.MinP == nil {
		return 0, false
	}
	return *e.MinP, true
}

// HasMinP returns true if min_p is configured.
func (e *ExecutionConfig) HasMinP() bool {
	return e != nil && e.MinP != nil
}

// GetRepetitionPenalty returns repetition_penalty and whether it was set.
func (e *ExecutionConfig) GetRepetitionPenalty() (float64, bool) {
	if e == nil || e.RepetitionPenalty == nil {
		return 0, false
	}
	return *e.RepetitionPenalty, true
}

// HasRepetitionPenalty returns true if repetition_penalty is configured.
func (e *ExecutionConfig) HasRepetitionPenalty() bool {
	return e != nil && e.RepetitionPenalty != nil
}

// GetSeed returns seed and whether it was set.
func (e *ExecutionConfig) GetSeed() (int, bool) {
	if e == nil || e.Seed == nil {
		return 0, false
	}
	return *e.Seed, true
}

// HasSeed returns true if seed is configured.
func (e *ExecutionConfig) HasSeed() bool {
	return e != nil && e.Seed != nil
}

// GetLogprobs returns logprobs and whether it was set.
func (e *ExecutionConfig) GetLogprobs() (int, bool) {
	if e == nil || e.Logprobs == nil {
		return 0, false
	}
	return *e.Logprobs, true
}

// HasLogprobs returns true if logprobs is configured.
func (e *ExecutionConfig) HasLogprobs() bool {
	return e != nil && e.Logprobs != nil
}

// GetStopTokenIDs returns stop_token_ids or nil.
func (e *ExecutionConfig) GetStopTokenIDs() []int {
	if e == nil {
		return nil
	}
	return e.StopTokenIDs
}

// HasStopTokenIDs returns true if stop_token_ids is configured.
func (e *ExecutionConfig) HasStopTokenIDs() bool {
	return e != nil && len(e.StopTokenIDs) > 0
}

// GetLogitBias returns logit_bias or nil.
func (e *ExecutionConfig) GetLogitBias() map[string]float64 {
	if e == nil {
		return nil
	}
	return e.LogitBias
}

// HasLogitBias returns true if logit_bias is configured.
func (e *ExecutionConfig) HasLogitBias() bool {
	return e != nil && len(e.LogitBias) > 0
}

// GetModality returns the modality string or empty.
func (e *ExecutionConfig) GetModality() string {
	if e == nil {
		return ""
	}
	return e.Modality
}

// HasModality returns true if modality is configured.
func (e *ExecutionConfig) HasModality() bool {
	return e != nil && e.Modality != ""
}

// GetImage returns the image config or nil.
func (e *ExecutionConfig) GetImage() *ImageConfig {
	if e == nil {
		return nil
	}
	return e.Image
}

// HasImage returns true if image config is configured.
func (e *ExecutionConfig) HasImage() bool {
	return e != nil && e.Image != nil
}

// GetAudio returns the audio config or nil.
func (e *ExecutionConfig) GetAudio() *AudioConfig {
	if e == nil {
		return nil
	}
	return e.Audio
}

// HasAudio returns true if audio config is configured.
func (e *ExecutionConfig) HasAudio() bool {
	return e != nil && e.Audio != nil
}

// GetEmbedding returns the embedding config or nil.
func (e *ExecutionConfig) GetEmbedding() *EmbeddingConfig {
	if e == nil {
		return nil
	}
	return e.Embedding
}

// HasEmbedding returns true if embedding config is configured.
func (e *ExecutionConfig) HasEmbedding() bool {
	return e != nil && e.Embedding != nil
}

// GetStreaming returns the streaming config or nil.
func (e *ExecutionConfig) GetStreaming() *StreamingConfig {
	if e == nil {
		return nil
	}
	return e.Streaming
}

// HasStreaming returns true if streaming config is configured.
func (e *ExecutionConfig) HasStreaming() bool {
	return e != nil && e.Streaming != nil
}

// GetAsync returns the async config or nil.
func (e *ExecutionConfig) GetAsync() *AsyncConfig {
	if e == nil {
		return nil
	}
	return e.Async
}

// HasAsync returns true if async config is configured.
func (e *ExecutionConfig) HasAsync() bool {
	return e != nil && e.Async != nil
}

// GetEffectiveProvider detects the intended provider from configuration.
// Returns the explicit provider if set, otherwise infers from config shape or model name.
func (e *ExecutionConfig) GetEffectiveProvider() string {
	if e == nil {
		return ""
	}

	// Explicit provider takes precedence
	if e.Provider != "" {
		return e.Provider
	}

	// Infer from configuration shape
	if e.GuidedDecoding != nil {
		return ProviderVLLM
	}
	if e.MinP != nil || e.RepetitionPenalty != nil || len(e.StopTokenIDs) > 0 {
		return ProviderVLLM
	}
	if e.Thinking != nil && e.Thinking.Enabled {
		return ProviderAnthropic
	}

	// Try to infer from model name
	if e.Model != "" {
		if isOpenAIModel(e.Model) {
			return ProviderOpenAI
		}
		if isAnthropicModel(e.Model) {
			return ProviderAnthropic
		}
		if isGeminiModel(e.Model) {
			return ProviderGemini
		}
		if isMistralModel(e.Model) {
			return ProviderMistral
		}
		if isCohereModel(e.Model) {
			return ProviderCohere
		}
	}

	return ""
}

// ToMap converts execution config to a parameter map for LLM clients.
// Only includes parameters that were explicitly set.
func (e *ExecutionConfig) ToMap() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}
	if e.MinP != nil {
		result[ParamKeyMinP] = *e.MinP
	}
	if e.RepetitionPenalty != nil {
		result[ParamKeyRepetitionPenalty] = *e.RepetitionPenalty
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}
	if e.Logprobs != nil {
		result[ParamKeyLogprobs] = *e.Logprobs
	}
	if len(e.StopTokenIDs) > 0 {
		result[ParamKeyStopTokenIDs] = e.StopTokenIDs
	}
	if len(e.LogitBias) > 0 {
		result[ParamKeyLogitBias] = e.LogitBias
	}

	// v2.5 media fields
	if e.Modality != "" {
		result[ParamKeyModality] = e.Modality
	}
	if e.Image != nil {
		result[ParamKeyImage] = e.Image.ToMap()
	}
	if e.Audio != nil {
		result[ParamKeyAudio] = e.Audio.ToMap()
	}
	if e.Embedding != nil {
		result[ParamKeyEmbedding] = e.Embedding.ToMap()
	}
	if e.Streaming != nil {
		result[ParamKeyStreaming] = e.Streaming.ToMap()
	}
	if e.Async != nil {
		result[ParamKeyAsync] = e.Async.ToMap()
	}

	return result
}

// ToOpenAI converts the execution config to OpenAI API format.
func (e *ExecutionConfig) ToOpenAI() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}

	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}
	if e.Logprobs != nil {
		result[ParamKeyLogprobs] = true
		result[ParamKeyTopLogprobs] = *e.Logprobs
	}
	if len(e.LogitBias) > 0 {
		result[ParamKeyLogitBias] = e.LogitBias
	}

	if e.ResponseFormat != nil {
		result[ParamKeyResponseFormat] = e.ResponseFormat.ToOpenAI()
	}

	// v2.5 OpenAI media params
	e.openAIImageParams(result)
	e.openAIAudioParams(result)
	e.openAIEmbeddingParams(result)
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// openAIImageParams adds OpenAI image generation params to the result map.
func (e *ExecutionConfig) openAIImageParams(result map[string]any) {
	if e.Image == nil {
		return
	}
	size := e.Image.EffectiveSize()
	if size != "" {
		result[ParamKeyImageSize] = size
	}
	if e.Image.Quality != "" {
		result[ParamKeyImageQuality] = e.Image.Quality
	}
	if e.Image.Style != "" {
		result[ParamKeyImageStyle] = e.Image.Style
	}
	if e.Image.NumImages != nil {
		result[ParamKeyImageN] = *e.Image.NumImages
	}
}

// openAIAudioParams adds OpenAI audio/TTS params to the result map.
func (e *ExecutionConfig) openAIAudioParams(result map[string]any) {
	if e.Audio == nil {
		return
	}
	if e.Audio.Voice != "" {
		result[ParamKeyVoice] = e.Audio.Voice
	}
	if e.Audio.Speed != nil {
		result[ParamKeySpeed] = *e.Audio.Speed
	}
	if e.Audio.OutputFormat != "" {
		// Only set response_format for audio if structured output response_format is not already set.
		// These target different OpenAI endpoints (TTS vs chat completions) but share the same key.
		if _, hasRF := result[ParamKeyResponseFormat]; !hasRF {
			result[ParamKeyResponseFormat] = e.Audio.OutputFormat
		}
	}
}

// openAIEmbeddingParams adds OpenAI embedding params to the result map.
func (e *ExecutionConfig) openAIEmbeddingParams(result map[string]any) {
	if e.Embedding == nil {
		return
	}
	if e.Embedding.Dimensions != nil {
		result[ParamKeyDimensions] = *e.Embedding.Dimensions
	}
	if e.Embedding.Format != "" {
		result[ParamKeyEncodingFormat] = e.Embedding.Format
	}
}

// ToAnthropic converts the execution config to Anthropic API format.
func (e *ExecutionConfig) ToAnthropic() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if e.TopK != nil {
		result[ParamKeyTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStopSequences] = e.StopSequences
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}

	// Handle extended thinking
	if e.Thinking != nil && e.Thinking.Enabled {
		thinking := map[string]any{
			ParamKeyThinkingType: ParamKeyThinkingTypeEnabled,
		}
		if e.Thinking.BudgetTokens != nil {
			thinking[ParamKeyBudgetTokens] = *e.Thinking.BudgetTokens
		}
		result[ParamKeyAnthropicThinking] = thinking
	}

	// Handle response format for Anthropic
	if e.ResponseFormat != nil {
		result[ParamKeyAnthropicOutputFormat] = e.ResponseFormat.ToAnthropic()
	}

	// v2.5: streaming only — no media generation params for Anthropic
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToGemini converts the execution config to Gemini/Vertex AI API format.
// Supports embedding parameters: output_dimensionality (from Dimensions) and task_type
// (from InputType via GeminiTaskType mapping). Also supports image params (aspectRatio,
// numberOfImages) and streaming.
func (e *ExecutionConfig) ToGemini() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}

	// Gemini uses generationConfig for parameters
	genConfig := make(map[string]any)
	if e.Temperature != nil {
		genConfig[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		genConfig[ParamKeyGeminiMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		genConfig[ParamKeyGeminiTopP] = *e.TopP
	}
	if e.TopK != nil {
		genConfig[ParamKeyGeminiTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		genConfig[ParamKeyGeminiStopSeqs] = e.StopSequences
	}

	if e.ResponseFormat != nil {
		genConfig[ParamKeyGeminiResponseMime] = GeminiResponseMimeJSON
		genConfig[ParamKeyGeminiResponseSchema] = e.ResponseFormat.ToGemini()
	}

	// v2.5 Gemini image params in generationConfig
	if e.Image != nil {
		if e.Image.AspectRatio != "" {
			genConfig[ParamKeyAspectRatio] = e.Image.AspectRatio
		}
		if e.Image.NumImages != nil {
			genConfig[ParamKeyGeminiNumImages] = *e.Image.NumImages
		}
	}

	// v2.7 Gemini embedding params in generationConfig
	if e.Embedding != nil {
		if e.Embedding.Dimensions != nil {
			genConfig[ParamKeyOutputDimensionality] = *e.Embedding.Dimensions
		}
		if e.Embedding.InputType != "" {
			if taskType, err := GeminiTaskType(e.Embedding.InputType); err == nil {
				genConfig[ParamKeyTaskType] = taskType
			}
		}
	}

	if len(genConfig) > 0 {
		result[ParamKeyGenerationConfig] = genConfig
	}

	// v2.5: streaming
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToVLLM converts the execution config to vLLM API format.
// Supports embedding parameters: normalize and pooling_type. Also supports guided decoding,
// extended inference params (min_p, repetition_penalty, logprobs, etc.), and streaming.
func (e *ExecutionConfig) ToVLLM() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if e.TopK != nil {
		result[ParamKeyTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}
	if e.MinP != nil {
		result[ParamKeyMinP] = *e.MinP
	}
	if e.RepetitionPenalty != nil {
		result[ParamKeyRepetitionPenalty] = *e.RepetitionPenalty
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}
	if e.Logprobs != nil {
		result[ParamKeyLogprobs] = *e.Logprobs
	}
	if len(e.StopTokenIDs) > 0 {
		result[ParamKeyStopTokenIDs] = e.StopTokenIDs
	}
	if len(e.LogitBias) > 0 {
		result[ParamKeyLogitBias] = e.LogitBias
	}

	// Add guided decoding parameters
	if e.GuidedDecoding != nil {
		gdParams := e.GuidedDecoding.ToVLLM()
		for k, v := range gdParams {
			result[k] = v
		}
	}

	// v2.7 vLLM embedding params
	if e.Embedding != nil {
		if e.Embedding.Normalize != nil {
			result[ParamKeyNormalize] = *e.Embedding.Normalize
		}
		if e.Embedding.PoolingType != "" {
			result[ParamKeyPoolingType] = e.Embedding.PoolingType
		}
	}

	// v2.5: streaming only — no media params for vLLM (text inference only)
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToMistral converts the execution config to Mistral AI API format.
// Mistral uses an OpenAI-compatible structure with provider-specific embedding params:
// output_dimension (from Dimensions), encoding_format (from Format), and output_dtype.
// Supports response_format and streaming.
func (e *ExecutionConfig) ToMistral() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}

	if e.ResponseFormat != nil {
		result[ParamKeyResponseFormat] = e.ResponseFormat.ToOpenAI()
	}

	// Mistral embedding params
	if e.Embedding != nil {
		if e.Embedding.Dimensions != nil {
			result[ParamKeyOutputDimension] = *e.Embedding.Dimensions
		}
		if e.Embedding.Format != "" {
			result[ParamKeyEncodingFormat] = e.Embedding.Format
		}
		if e.Embedding.OutputDtype != "" {
			result[ParamKeyOutputDtype] = e.Embedding.OutputDtype
		}
	}

	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToCohere converts the execution config to Cohere API format.
// Cohere uses different parameter names than OpenAI: "p" for top_p, "k" for top_k,
// "stop_sequences" for stop. Embedding params: output_dimension, input_type,
// embedding_types (OutputDtype as []string), and truncate (truncation in UPPER_CASE via
// CohereUpperCase). Supports streaming.
func (e *ExecutionConfig) ToCohere() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyCohereTopP] = *e.TopP
	}
	if e.TopK != nil {
		result[ParamKeyCohereTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStopSequences] = e.StopSequences
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}

	// Cohere embedding params
	if e.Embedding != nil {
		if e.Embedding.Dimensions != nil {
			result[ParamKeyOutputDimension] = *e.Embedding.Dimensions
		}
		if e.Embedding.InputType != "" {
			result[ParamKeyInputType] = e.Embedding.InputType
		}
		if e.Embedding.OutputDtype != "" {
			result[ParamKeyEmbeddingTypes] = []string{e.Embedding.OutputDtype}
		}
		if e.Embedding.Truncation != "" {
			if upper, err := CohereUpperCase(e.Embedding.Truncation); err == nil {
				result[ParamKeyTruncate] = upper
			}
		}
	}

	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ProviderFormat returns the response format for a specific provider.
func (e *ExecutionConfig) ProviderFormat(provider string) (map[string]any, error) {
	if e == nil {
		return nil, nil
	}

	switch provider {
	case ProviderOpenAI, ProviderAzure:
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToOpenAI(), nil
		}
		return nil, nil

	case ProviderAnthropic:
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToAnthropic(), nil
		}
		return nil, nil

	case ProviderGoogle, ProviderGemini, ProviderVertex:
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToGemini(), nil
		}
		return nil, nil

	case ProviderVLLM:
		if e.GuidedDecoding != nil {
			return e.GuidedDecoding.ToVLLM(), nil
		}
		return nil, nil

	case ProviderMistral:
		// Mistral uses OpenAI-compatible response_format
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToOpenAI(), nil
		}
		return nil, nil

	case ProviderCohere:
		// Cohere does not use response_format
		return nil, nil

	default:
		return nil, NewSchemaProviderError(ErrMsgSchemaUnsupportedProvider, provider)
	}
}

// Merge creates a new ExecutionConfig that merges other into the receiver.
// The other config's non-nil/non-zero values override the receiver's values (more-specific wins).
// Returns a new config; neither the receiver nor other is modified.
//
// This implements 3-layer precedence for agent compilation:
//
//	agent definition (base) → skill override (resolved) → runtime override (SkillRef.Execution)
//
// Each layer is merged left-to-right: base.Merge(skillOverride).Merge(runtimeOverride).
// For each field, the rightmost non-nil/non-zero value wins.
//
// Example:
//
//	agent := &ExecutionConfig{Provider: "openai", Model: "gpt-4", Temperature: floatPtr(0.7)}
//	skill := &ExecutionConfig{Temperature: floatPtr(0.1)}
//	effective := agent.Merge(skill) // Provider: "openai", Model: "gpt-4", Temperature: 0.1
func (e *ExecutionConfig) Merge(other *ExecutionConfig) *ExecutionConfig {
	if e == nil && other == nil {
		return nil
	}
	if e == nil {
		return other.Clone()
	}
	if other == nil {
		return e.Clone()
	}

	result := e.Clone()

	// Scalar overrides
	if other.Provider != "" {
		result.Provider = other.Provider
	}
	if other.Model != "" {
		result.Model = other.Model
	}

	// Pointer overrides
	result.Temperature = coalesceFloat64Ptr(other.Temperature, result.Temperature)
	result.MaxTokens = coalesceIntPtr(other.MaxTokens, result.MaxTokens)
	result.TopP = coalesceFloat64Ptr(other.TopP, result.TopP)
	result.TopK = coalesceIntPtr(other.TopK, result.TopK)

	if len(other.StopSequences) > 0 {
		result.StopSequences = make([]string, len(other.StopSequences))
		copy(result.StopSequences, other.StopSequences)
	}

	result.MinP = coalesceFloat64Ptr(other.MinP, result.MinP)
	result.RepetitionPenalty = coalesceFloat64Ptr(other.RepetitionPenalty, result.RepetitionPenalty)
	result.Seed = coalesceIntPtr(other.Seed, result.Seed)
	result.Logprobs = coalesceIntPtr(other.Logprobs, result.Logprobs)

	if len(other.StopTokenIDs) > 0 {
		result.StopTokenIDs = make([]int, len(other.StopTokenIDs))
		copy(result.StopTokenIDs, other.StopTokenIDs)
	}
	if len(other.LogitBias) > 0 {
		result.LogitBias = make(map[string]float64, len(other.LogitBias))
		for k, v := range other.LogitBias {
			result.LogitBias[k] = v
		}
	}

	if other.Thinking != nil {
		result.Thinking = &ThinkingConfig{Enabled: other.Thinking.Enabled}
		if other.Thinking.BudgetTokens != nil {
			bt := *other.Thinking.BudgetTokens
			result.Thinking.BudgetTokens = &bt
		}
	}

	if other.ResponseFormat != nil {
		result.ResponseFormat = cloneResponseFormat(other.ResponseFormat)
	}
	if other.GuidedDecoding != nil {
		result.GuidedDecoding = cloneGuidedDecoding(other.GuidedDecoding)
	}

	// v2.5 media fields
	if other.Modality != "" {
		result.Modality = other.Modality
	}
	if other.Image != nil {
		result.Image = other.Image.Clone()
	}
	if other.Audio != nil {
		result.Audio = other.Audio.Clone()
	}
	if other.Embedding != nil {
		result.Embedding = other.Embedding.Clone()
	}
	if other.Streaming != nil {
		result.Streaming = other.Streaming.Clone()
	}
	if other.Async != nil {
		result.Async = other.Async.Clone()
	}

	// Merge provider options (other wins on conflict)
	if len(other.ProviderOptions) > 0 {
		if result.ProviderOptions == nil {
			result.ProviderOptions = make(map[string]any, len(other.ProviderOptions))
		}
		for k, v := range other.ProviderOptions {
			result.ProviderOptions[k] = v
		}
	}

	return result
}

// coalesceFloat64Ptr returns the first non-nil pointer.
func coalesceFloat64Ptr(a, b *float64) *float64 {
	if a != nil {
		v := *a
		return &v
	}
	return b
}

// coalesceIntPtr returns the first non-nil pointer.
func coalesceIntPtr(a, b *int) *int {
	if a != nil {
		v := *a
		return &v
	}
	return b
}

// JSON returns the JSON representation of the execution config.
func (e *ExecutionConfig) JSON() (string, error) {
	if e == nil {
		return "", nil
	}
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// YAML returns the YAML representation of the execution config.
func (e *ExecutionConfig) YAML() (string, error) {
	if e == nil {
		return "", nil
	}
	data, err := yaml.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
