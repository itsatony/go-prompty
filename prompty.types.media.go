package prompty

import "fmt"

// ImageConfig configures image generation parameters.
type ImageConfig struct {
	// Width of the generated image in pixels (1-8192)
	Width *int `yaml:"width,omitempty" json:"width,omitempty"`
	// Height of the generated image in pixels (1-8192)
	Height *int `yaml:"height,omitempty" json:"height,omitempty"`
	// Size is a provider-specific size string (e.g., "1024x1024")
	Size string `yaml:"size,omitempty" json:"size,omitempty"`
	// Quality of the generated image: "standard", "hd", "low", "medium", "high"
	Quality string `yaml:"quality,omitempty" json:"quality,omitempty"`
	// Style of the generated image: "natural", "vivid"
	Style string `yaml:"style,omitempty" json:"style,omitempty"`
	// AspectRatio for the generated image (e.g., "16:9", "1:1")
	AspectRatio string `yaml:"aspect_ratio,omitempty" json:"aspect_ratio,omitempty"`
	// NegativePrompt describes content to avoid in generation
	NegativePrompt string `yaml:"negative_prompt,omitempty" json:"negative_prompt,omitempty"`
	// NumImages is the number of images to generate (1-10)
	NumImages *int `yaml:"num_images,omitempty" json:"num_images,omitempty"`
	// GuidanceScale controls adherence to the prompt (0.0-30.0)
	GuidanceScale *float64 `yaml:"guidance_scale,omitempty" json:"guidance_scale,omitempty"`
	// Steps is the number of diffusion steps (1-200)
	Steps *int `yaml:"steps,omitempty" json:"steps,omitempty"`
	// Strength controls how much to transform a reference image (0.0-1.0)
	Strength *float64 `yaml:"strength,omitempty" json:"strength,omitempty"`
}

// Validate checks the image config for consistency.
func (c *ImageConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.Width != nil {
		if *c.Width < 1 || *c.Width > ImageMaxWidth {
			return NewPromptValidationError(ErrMsgImageWidthOutOfRange, "")
		}
	}

	if c.Height != nil {
		if *c.Height < 1 || *c.Height > ImageMaxHeight {
			return NewPromptValidationError(ErrMsgImageHeightOutOfRange, "")
		}
	}

	if c.Quality != "" && !isValidImageQuality(c.Quality) {
		return NewPromptValidationError(ErrMsgImageInvalidQuality, "")
	}

	if c.Style != "" && !isValidImageStyle(c.Style) {
		return NewPromptValidationError(ErrMsgImageInvalidStyle, "")
	}

	if c.NumImages != nil {
		if *c.NumImages < 1 || *c.NumImages > ImageMaxNumImages {
			return NewPromptValidationError(ErrMsgImageNumImagesOutOfRange, "")
		}
	}

	if c.GuidanceScale != nil {
		if *c.GuidanceScale < 0.0 || *c.GuidanceScale > ImageMaxGuidanceScale {
			return NewPromptValidationError(ErrMsgImageGuidanceScaleOutOfRange, "")
		}
	}

	if c.Steps != nil {
		if *c.Steps < 1 || *c.Steps > ImageMaxSteps {
			return NewPromptValidationError(ErrMsgImageStepsOutOfRange, "")
		}
	}

	if c.Strength != nil {
		if *c.Strength < 0.0 || *c.Strength > 1.0 {
			return NewPromptValidationError(ErrMsgImageStrengthOutOfRange, "")
		}
	}

	return nil
}

// Clone creates a deep copy of the image config.
func (c *ImageConfig) Clone() *ImageConfig {
	if c == nil {
		return nil
	}

	clone := &ImageConfig{
		Size:           c.Size,
		Quality:        c.Quality,
		Style:          c.Style,
		AspectRatio:    c.AspectRatio,
		NegativePrompt: c.NegativePrompt,
	}

	if c.Width != nil {
		v := *c.Width
		clone.Width = &v
	}
	if c.Height != nil {
		v := *c.Height
		clone.Height = &v
	}
	if c.NumImages != nil {
		v := *c.NumImages
		clone.NumImages = &v
	}
	if c.GuidanceScale != nil {
		v := *c.GuidanceScale
		clone.GuidanceScale = &v
	}
	if c.Steps != nil {
		v := *c.Steps
		clone.Steps = &v
	}
	if c.Strength != nil {
		v := *c.Strength
		clone.Strength = &v
	}

	return clone
}

// ToMap converts the image config to a parameter map.
func (c *ImageConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Width != nil {
		result[ParamKeyWidth] = *c.Width
	}
	if c.Height != nil {
		result[ParamKeyHeight] = *c.Height
	}
	if c.Size != "" {
		result[ParamKeyImageSize] = c.Size
	}
	if c.Quality != "" {
		result[ParamKeyImageQuality] = c.Quality
	}
	if c.Style != "" {
		result[ParamKeyImageStyle] = c.Style
	}
	if c.AspectRatio != "" {
		result[ParamKeyAspectRatio] = c.AspectRatio
	}
	if c.NegativePrompt != "" {
		result[ParamKeyNegativePrompt] = c.NegativePrompt
	}
	if c.NumImages != nil {
		result[ParamKeyNumImages] = *c.NumImages
	}
	if c.GuidanceScale != nil {
		result[ParamKeyGuidanceScale] = *c.GuidanceScale
	}
	if c.Steps != nil {
		result[ParamKeySteps] = *c.Steps
	}
	if c.Strength != nil {
		result[ParamKeyStrength] = *c.Strength
	}

	return result
}

// EffectiveSize returns the image size string. If Size is set, it is returned directly.
// Otherwise, if both Width and Height are set, a "WxH" string is derived.
func (c *ImageConfig) EffectiveSize() string {
	if c == nil {
		return ""
	}
	if c.Size != "" {
		return c.Size
	}
	if c.Width != nil && c.Height != nil {
		return fmt.Sprintf("%dx%d", *c.Width, *c.Height)
	}
	return ""
}

// AudioConfig configures audio generation (TTS/transcription) parameters.
type AudioConfig struct {
	// Voice is the voice name (e.g., "alloy", "echo", "nova")
	Voice string `yaml:"voice,omitempty" json:"voice,omitempty"`
	// VoiceID is a provider-specific voice identifier
	VoiceID string `yaml:"voice_id,omitempty" json:"voice_id,omitempty"`
	// Speed controls the playback speed (0.25-4.0)
	Speed *float64 `yaml:"speed,omitempty" json:"speed,omitempty"`
	// OutputFormat is the audio output format: "mp3", "opus", "aac", "flac", "wav", "pcm"
	OutputFormat string `yaml:"output_format,omitempty" json:"output_format,omitempty"`
	// Duration is the maximum duration in seconds (0-600)
	Duration *float64 `yaml:"duration,omitempty" json:"duration,omitempty"`
	// Language is the language code (e.g., "en", "es")
	Language string `yaml:"language,omitempty" json:"language,omitempty"`
}

// Validate checks the audio config for consistency.
func (c *AudioConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.Speed != nil {
		if *c.Speed < AudioMinSpeed || *c.Speed > AudioMaxSpeed {
			return NewPromptValidationError(ErrMsgAudioSpeedOutOfRange, "")
		}
	}

	if c.OutputFormat != "" && !isValidAudioFormat(c.OutputFormat) {
		return NewPromptValidationError(ErrMsgAudioInvalidFormat, "")
	}

	if c.Duration != nil {
		if *c.Duration <= 0.0 || *c.Duration > AudioMaxDuration {
			return NewPromptValidationError(ErrMsgAudioDurationOutOfRange, "")
		}
	}

	return nil
}

// Clone creates a deep copy of the audio config.
func (c *AudioConfig) Clone() *AudioConfig {
	if c == nil {
		return nil
	}

	clone := &AudioConfig{
		Voice:        c.Voice,
		VoiceID:      c.VoiceID,
		OutputFormat: c.OutputFormat,
		Language:     c.Language,
	}

	if c.Speed != nil {
		v := *c.Speed
		clone.Speed = &v
	}
	if c.Duration != nil {
		v := *c.Duration
		clone.Duration = &v
	}

	return clone
}

// ToMap converts the audio config to a parameter map.
func (c *AudioConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Voice != "" {
		result[ParamKeyVoice] = c.Voice
	}
	if c.VoiceID != "" {
		result[ParamKeyVoiceID] = c.VoiceID
	}
	if c.Speed != nil {
		result[ParamKeySpeed] = *c.Speed
	}
	if c.OutputFormat != "" {
		result[ParamKeyOutputFormat] = c.OutputFormat
	}
	if c.Duration != nil {
		result[ParamKeyDuration] = *c.Duration
	}
	if c.Language != "" {
		result[ParamKeyLanguage] = c.Language
	}

	return result
}

// EmbeddingConfig configures embedding generation parameters.
type EmbeddingConfig struct {
	// Dimensions is the number of embedding dimensions (1-65536)
	Dimensions *int `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	// Format is the embedding output format: "float" or "base64"
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
}

// Validate checks the embedding config for consistency.
func (c *EmbeddingConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.Dimensions != nil {
		if *c.Dimensions < 1 || *c.Dimensions > EmbeddingMaxDimensions {
			return NewPromptValidationError(ErrMsgEmbeddingDimensionsOutOfRange, "")
		}
	}

	if c.Format != "" && !isValidEmbeddingFormat(c.Format) {
		return NewPromptValidationError(ErrMsgEmbeddingInvalidFormat, "")
	}

	return nil
}

// Clone creates a deep copy of the embedding config.
func (c *EmbeddingConfig) Clone() *EmbeddingConfig {
	if c == nil {
		return nil
	}

	clone := &EmbeddingConfig{
		Format: c.Format,
	}

	if c.Dimensions != nil {
		v := *c.Dimensions
		clone.Dimensions = &v
	}

	return clone
}

// ToMap converts the embedding config to a parameter map.
func (c *EmbeddingConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Dimensions != nil {
		result[ParamKeyDimensions] = *c.Dimensions
	}
	if c.Format != "" {
		result[ParamKeyEncodingFormat] = c.Format
	}

	return result
}

// AsyncConfig configures asynchronous execution behavior.
type AsyncConfig struct {
	// Enabled indicates whether async execution is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`
	// PollIntervalSeconds is the polling interval in seconds (must be > 0)
	PollIntervalSeconds *float64 `yaml:"poll_interval_seconds,omitempty" json:"poll_interval_seconds,omitempty"`
	// PollTimeoutSeconds is the maximum polling duration in seconds (must be > 0, >= PollInterval)
	PollTimeoutSeconds *float64 `yaml:"poll_timeout_seconds,omitempty" json:"poll_timeout_seconds,omitempty"`
}

// Validate checks the async config for consistency.
func (c *AsyncConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.PollIntervalSeconds != nil {
		if *c.PollIntervalSeconds <= 0 {
			return NewPromptValidationError(ErrMsgAsyncPollIntervalInvalid, "")
		}
	}

	if c.PollTimeoutSeconds != nil {
		if *c.PollTimeoutSeconds <= 0 {
			return NewPromptValidationError(ErrMsgAsyncPollTimeoutInvalid, "")
		}
	}

	if c.PollIntervalSeconds != nil && c.PollTimeoutSeconds != nil {
		if *c.PollTimeoutSeconds < *c.PollIntervalSeconds {
			return NewPromptValidationError(ErrMsgAsyncPollTimeoutTooSmall, "")
		}
	}

	return nil
}

// Clone creates a deep copy of the async config.
func (c *AsyncConfig) Clone() *AsyncConfig {
	if c == nil {
		return nil
	}

	clone := &AsyncConfig{
		Enabled: c.Enabled,
	}

	if c.PollIntervalSeconds != nil {
		v := *c.PollIntervalSeconds
		clone.PollIntervalSeconds = &v
	}
	if c.PollTimeoutSeconds != nil {
		v := *c.PollTimeoutSeconds
		clone.PollTimeoutSeconds = &v
	}

	return clone
}

// ToMap converts the async config to a parameter map.
func (c *AsyncConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	result[ParamKeyEnabled] = c.Enabled
	if c.PollIntervalSeconds != nil {
		result[ParamKeyPollInterval] = *c.PollIntervalSeconds
	}
	if c.PollTimeoutSeconds != nil {
		result[ParamKeyPollTimeout] = *c.PollTimeoutSeconds
	}

	return result
}

// isValidModality checks if the given string is a valid modality value.
func isValidModality(m string) bool {
	switch m {
	case ModalityText, ModalityImage, ModalityAudioSpeech,
		ModalityAudioTranscription, ModalityMusic,
		ModalitySoundEffects, ModalityEmbedding:
		return true
	default:
		return false
	}
}

// isValidImageQuality checks if the given string is a valid image quality.
func isValidImageQuality(q string) bool {
	switch q {
	case ImageQualityStandard, ImageQualityHD,
		ImageQualityLow, ImageQualityMedium, ImageQualityHigh:
		return true
	default:
		return false
	}
}

// isValidImageStyle checks if the given string is a valid image style.
func isValidImageStyle(s string) bool {
	switch s {
	case ImageStyleNatural, ImageStyleVivid:
		return true
	default:
		return false
	}
}

// isValidAudioFormat checks if the given string is a valid audio format.
func isValidAudioFormat(f string) bool {
	switch f {
	case AudioFormatMP3, AudioFormatOpus, AudioFormatAAC,
		AudioFormatFLAC, AudioFormatWAV, AudioFormatPCM:
		return true
	default:
		return false
	}
}

// isValidEmbeddingFormat checks if the given string is a valid embedding format.
func isValidEmbeddingFormat(f string) bool {
	switch f {
	case EmbeddingFormatFloat, EmbeddingFormatBase64:
		return true
	default:
		return false
	}
}

// isValidStreamMethod checks if the given string is a valid streaming method.
func isValidStreamMethod(m string) bool {
	switch m {
	case StreamMethodSSE, StreamMethodWebSocket:
		return true
	default:
		return false
	}
}
