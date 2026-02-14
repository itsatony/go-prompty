package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ImageConfig.Validate ---

func TestImageConfig_Validate(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	floatPtr := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		config  *ImageConfig
		wantErr bool
		errMsg  string
	}{
		{name: "nil config", config: nil, wantErr: false},
		{name: "empty config", config: &ImageConfig{}, wantErr: false},
		{name: "valid width and height", config: &ImageConfig{Width: intPtr(1024), Height: intPtr(1024)}, wantErr: false},
		{name: "min width", config: &ImageConfig{Width: intPtr(1)}, wantErr: false},
		{name: "max width", config: &ImageConfig{Width: intPtr(ImageMaxWidth)}, wantErr: false},
		{name: "width zero", config: &ImageConfig{Width: intPtr(0)}, wantErr: true, errMsg: ErrMsgImageWidthOutOfRange},
		{name: "width negative", config: &ImageConfig{Width: intPtr(-1)}, wantErr: true, errMsg: ErrMsgImageWidthOutOfRange},
		{name: "width too large", config: &ImageConfig{Width: intPtr(ImageMaxWidth + 1)}, wantErr: true, errMsg: ErrMsgImageWidthOutOfRange},
		{name: "height zero", config: &ImageConfig{Height: intPtr(0)}, wantErr: true, errMsg: ErrMsgImageHeightOutOfRange},
		{name: "height too large", config: &ImageConfig{Height: intPtr(ImageMaxHeight + 1)}, wantErr: true, errMsg: ErrMsgImageHeightOutOfRange},
		{name: "valid quality standard", config: &ImageConfig{Quality: ImageQualityStandard}, wantErr: false},
		{name: "valid quality hd", config: &ImageConfig{Quality: ImageQualityHD}, wantErr: false},
		{name: "valid quality low", config: &ImageConfig{Quality: ImageQualityLow}, wantErr: false},
		{name: "valid quality medium", config: &ImageConfig{Quality: ImageQualityMedium}, wantErr: false},
		{name: "valid quality high", config: &ImageConfig{Quality: ImageQualityHigh}, wantErr: false},
		{name: "invalid quality", config: &ImageConfig{Quality: "ultra"}, wantErr: true, errMsg: ErrMsgImageInvalidQuality},
		{name: "valid style natural", config: &ImageConfig{Style: ImageStyleNatural}, wantErr: false},
		{name: "valid style vivid", config: &ImageConfig{Style: ImageStyleVivid}, wantErr: false},
		{name: "invalid style", config: &ImageConfig{Style: "abstract"}, wantErr: true, errMsg: ErrMsgImageInvalidStyle},
		{name: "valid num_images", config: &ImageConfig{NumImages: intPtr(5)}, wantErr: false},
		{name: "num_images min", config: &ImageConfig{NumImages: intPtr(1)}, wantErr: false},
		{name: "num_images max", config: &ImageConfig{NumImages: intPtr(ImageMaxNumImages)}, wantErr: false},
		{name: "num_images zero", config: &ImageConfig{NumImages: intPtr(0)}, wantErr: true, errMsg: ErrMsgImageNumImagesOutOfRange},
		{name: "num_images too large", config: &ImageConfig{NumImages: intPtr(ImageMaxNumImages + 1)}, wantErr: true, errMsg: ErrMsgImageNumImagesOutOfRange},
		{name: "valid guidance_scale", config: &ImageConfig{GuidanceScale: floatPtr(7.5)}, wantErr: false},
		{name: "guidance_scale zero", config: &ImageConfig{GuidanceScale: floatPtr(0.0)}, wantErr: false},
		{name: "guidance_scale max", config: &ImageConfig{GuidanceScale: floatPtr(ImageMaxGuidanceScale)}, wantErr: false},
		{name: "guidance_scale negative", config: &ImageConfig{GuidanceScale: floatPtr(-0.1)}, wantErr: true, errMsg: ErrMsgImageGuidanceScaleOutOfRange},
		{name: "guidance_scale too high", config: &ImageConfig{GuidanceScale: floatPtr(ImageMaxGuidanceScale + 0.1)}, wantErr: true, errMsg: ErrMsgImageGuidanceScaleOutOfRange},
		{name: "valid steps", config: &ImageConfig{Steps: intPtr(50)}, wantErr: false},
		{name: "steps min", config: &ImageConfig{Steps: intPtr(1)}, wantErr: false},
		{name: "steps max", config: &ImageConfig{Steps: intPtr(ImageMaxSteps)}, wantErr: false},
		{name: "steps zero", config: &ImageConfig{Steps: intPtr(0)}, wantErr: true, errMsg: ErrMsgImageStepsOutOfRange},
		{name: "steps too large", config: &ImageConfig{Steps: intPtr(ImageMaxSteps + 1)}, wantErr: true, errMsg: ErrMsgImageStepsOutOfRange},
		{name: "valid strength", config: &ImageConfig{Strength: floatPtr(0.5)}, wantErr: false},
		{name: "strength zero", config: &ImageConfig{Strength: floatPtr(0.0)}, wantErr: false},
		{name: "strength one", config: &ImageConfig{Strength: floatPtr(1.0)}, wantErr: false},
		{name: "strength negative", config: &ImageConfig{Strength: floatPtr(-0.1)}, wantErr: true, errMsg: ErrMsgImageStrengthOutOfRange},
		{name: "strength too high", config: &ImageConfig{Strength: floatPtr(1.1)}, wantErr: true, errMsg: ErrMsgImageStrengthOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- ImageConfig.Clone ---

func TestImageConfig_Clone(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	floatPtr := func(v float64) *float64 { return &v }

	t.Run("nil clone", func(t *testing.T) {
		var c *ImageConfig
		assert.Nil(t, c.Clone())
	})

	t.Run("deep copy", func(t *testing.T) {
		original := &ImageConfig{
			Width:          intPtr(1024),
			Height:         intPtr(768),
			Size:           "1024x768",
			Quality:        ImageQualityHD,
			Style:          ImageStyleVivid,
			AspectRatio:    "16:9",
			NegativePrompt: "blurry",
			NumImages:      intPtr(3),
			GuidanceScale:  floatPtr(7.5),
			Steps:          intPtr(50),
			Strength:       floatPtr(0.8),
		}

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.Equal(t, *original.Width, *clone.Width)
		assert.Equal(t, *original.Height, *clone.Height)
		assert.Equal(t, original.Size, clone.Size)
		assert.Equal(t, original.Quality, clone.Quality)
		assert.Equal(t, original.Style, clone.Style)
		assert.Equal(t, original.AspectRatio, clone.AspectRatio)
		assert.Equal(t, original.NegativePrompt, clone.NegativePrompt)
		assert.Equal(t, *original.NumImages, *clone.NumImages)
		assert.Equal(t, *original.GuidanceScale, *clone.GuidanceScale)
		assert.Equal(t, *original.Steps, *clone.Steps)
		assert.Equal(t, *original.Strength, *clone.Strength)

		// Verify independence
		*clone.Width = 512
		assert.NotEqual(t, *original.Width, *clone.Width)
		*clone.GuidanceScale = 1.0
		assert.NotEqual(t, *original.GuidanceScale, *clone.GuidanceScale)
	})
}

// --- ImageConfig.EffectiveSize ---

func TestImageConfig_EffectiveSize(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name   string
		config *ImageConfig
		want   string
	}{
		{name: "nil config", config: nil, want: ""},
		{name: "explicit size", config: &ImageConfig{Size: "1024x1024"}, want: "1024x1024"},
		{name: "explicit size overrides w/h", config: &ImageConfig{Size: "1024x1024", Width: intPtr(512), Height: intPtr(512)}, want: "1024x1024"},
		{name: "derived from w/h", config: &ImageConfig{Width: intPtr(1024), Height: intPtr(768)}, want: "1024x768"},
		{name: "width only", config: &ImageConfig{Width: intPtr(1024)}, want: ""},
		{name: "height only", config: &ImageConfig{Height: intPtr(768)}, want: ""},
		{name: "empty config", config: &ImageConfig{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.EffectiveSize())
		})
	}
}

// --- ImageConfig.ToMap ---

func TestImageConfig_ToMap(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	floatPtr := func(v float64) *float64 { return &v }

	t.Run("nil", func(t *testing.T) {
		var c *ImageConfig
		assert.Nil(t, c.ToMap())
	})

	t.Run("all fields", func(t *testing.T) {
		c := &ImageConfig{
			Width:          intPtr(1024),
			Height:         intPtr(768),
			Size:           "1024x768",
			Quality:        ImageQualityHD,
			Style:          ImageStyleVivid,
			AspectRatio:    "16:9",
			NegativePrompt: "blurry",
			NumImages:      intPtr(3),
			GuidanceScale:  floatPtr(7.5),
			Steps:          intPtr(50),
			Strength:       floatPtr(0.8),
		}

		m := c.ToMap()
		assert.Equal(t, 1024, m[ParamKeyWidth])
		assert.Equal(t, 768, m[ParamKeyHeight])
		assert.Equal(t, "1024x768", m[ParamKeyImageSize])
		assert.Equal(t, ImageQualityHD, m[ParamKeyImageQuality])
		assert.Equal(t, ImageStyleVivid, m[ParamKeyImageStyle])
		assert.Equal(t, "16:9", m[ParamKeyAspectRatio])
		assert.Equal(t, "blurry", m[ParamKeyNegativePrompt])
		assert.Equal(t, 3, m[ParamKeyNumImages])
		assert.Equal(t, 7.5, m[ParamKeyGuidanceScale])
		assert.Equal(t, 50, m[ParamKeySteps])
		assert.Equal(t, 0.8, m[ParamKeyStrength])
	})
}

// --- AudioConfig.Validate ---

func TestAudioConfig_Validate(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		config  *AudioConfig
		wantErr bool
		errMsg  string
	}{
		{name: "nil config", config: nil, wantErr: false},
		{name: "empty config", config: &AudioConfig{}, wantErr: false},
		{name: "valid speed", config: &AudioConfig{Speed: floatPtr(1.0)}, wantErr: false},
		{name: "speed min", config: &AudioConfig{Speed: floatPtr(AudioMinSpeed)}, wantErr: false},
		{name: "speed max", config: &AudioConfig{Speed: floatPtr(AudioMaxSpeed)}, wantErr: false},
		{name: "speed too low", config: &AudioConfig{Speed: floatPtr(0.1)}, wantErr: true, errMsg: ErrMsgAudioSpeedOutOfRange},
		{name: "speed too high", config: &AudioConfig{Speed: floatPtr(4.1)}, wantErr: true, errMsg: ErrMsgAudioSpeedOutOfRange},
		{name: "valid format mp3", config: &AudioConfig{OutputFormat: AudioFormatMP3}, wantErr: false},
		{name: "valid format opus", config: &AudioConfig{OutputFormat: AudioFormatOpus}, wantErr: false},
		{name: "valid format aac", config: &AudioConfig{OutputFormat: AudioFormatAAC}, wantErr: false},
		{name: "valid format flac", config: &AudioConfig{OutputFormat: AudioFormatFLAC}, wantErr: false},
		{name: "valid format wav", config: &AudioConfig{OutputFormat: AudioFormatWAV}, wantErr: false},
		{name: "valid format pcm", config: &AudioConfig{OutputFormat: AudioFormatPCM}, wantErr: false},
		{name: "invalid format", config: &AudioConfig{OutputFormat: "ogg"}, wantErr: true, errMsg: ErrMsgAudioInvalidFormat},
		{name: "valid duration", config: &AudioConfig{Duration: floatPtr(120.0)}, wantErr: false},
		{name: "duration zero", config: &AudioConfig{Duration: floatPtr(0.0)}, wantErr: true, errMsg: ErrMsgAudioDurationOutOfRange},
		{name: "duration negative", config: &AudioConfig{Duration: floatPtr(-1.0)}, wantErr: true, errMsg: ErrMsgAudioDurationOutOfRange},
		{name: "duration max", config: &AudioConfig{Duration: floatPtr(AudioMaxDuration)}, wantErr: false},
		{name: "duration too high", config: &AudioConfig{Duration: floatPtr(AudioMaxDuration + 1)}, wantErr: true, errMsg: ErrMsgAudioDurationOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- AudioConfig.Clone ---

func TestAudioConfig_Clone(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }

	t.Run("nil clone", func(t *testing.T) {
		var c *AudioConfig
		assert.Nil(t, c.Clone())
	})

	t.Run("deep copy", func(t *testing.T) {
		original := &AudioConfig{
			Voice:        "alloy",
			VoiceID:      "voice_123",
			Speed:        floatPtr(1.5),
			OutputFormat: AudioFormatMP3,
			Duration:     floatPtr(30.0),
			Language:     "en",
		}

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.Equal(t, original.Voice, clone.Voice)
		assert.Equal(t, original.VoiceID, clone.VoiceID)
		assert.Equal(t, *original.Speed, *clone.Speed)
		assert.Equal(t, original.OutputFormat, clone.OutputFormat)
		assert.Equal(t, *original.Duration, *clone.Duration)
		assert.Equal(t, original.Language, clone.Language)

		// Verify independence
		*clone.Speed = 2.0
		assert.NotEqual(t, *original.Speed, *clone.Speed)
	})
}

// --- AudioConfig.ToMap ---

func TestAudioConfig_ToMap(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }

	t.Run("nil", func(t *testing.T) {
		var c *AudioConfig
		assert.Nil(t, c.ToMap())
	})

	t.Run("all fields", func(t *testing.T) {
		c := &AudioConfig{
			Voice:        "alloy",
			VoiceID:      "voice_123",
			Speed:        floatPtr(1.5),
			OutputFormat: AudioFormatMP3,
			Duration:     floatPtr(30.0),
			Language:     "en",
		}

		m := c.ToMap()
		assert.Equal(t, "alloy", m[ParamKeyVoice])
		assert.Equal(t, "voice_123", m[ParamKeyVoiceID])
		assert.Equal(t, 1.5, m[ParamKeySpeed])
		assert.Equal(t, AudioFormatMP3, m[ParamKeyOutputFormat])
		assert.Equal(t, 30.0, m[ParamKeyDuration])
		assert.Equal(t, "en", m[ParamKeyLanguage])
	})
}

// --- EmbeddingConfig.Validate ---

func TestEmbeddingConfig_Validate(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name    string
		config  *EmbeddingConfig
		wantErr bool
		errMsg  string
	}{
		{name: "nil config", config: nil, wantErr: false},
		{name: "empty config", config: &EmbeddingConfig{}, wantErr: false},
		{name: "valid dimensions", config: &EmbeddingConfig{Dimensions: intPtr(1536)}, wantErr: false},
		{name: "dimensions min", config: &EmbeddingConfig{Dimensions: intPtr(1)}, wantErr: false},
		{name: "dimensions max", config: &EmbeddingConfig{Dimensions: intPtr(EmbeddingMaxDimensions)}, wantErr: false},
		{name: "dimensions zero", config: &EmbeddingConfig{Dimensions: intPtr(0)}, wantErr: true, errMsg: ErrMsgEmbeddingDimensionsOutOfRange},
		{name: "dimensions too large", config: &EmbeddingConfig{Dimensions: intPtr(EmbeddingMaxDimensions + 1)}, wantErr: true, errMsg: ErrMsgEmbeddingDimensionsOutOfRange},
		{name: "valid format float", config: &EmbeddingConfig{Format: EmbeddingFormatFloat}, wantErr: false},
		{name: "valid format base64", config: &EmbeddingConfig{Format: EmbeddingFormatBase64}, wantErr: false},
		{name: "invalid format", config: &EmbeddingConfig{Format: "binary"}, wantErr: true, errMsg: ErrMsgEmbeddingInvalidFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- EmbeddingConfig.Clone ---

func TestEmbeddingConfig_Clone(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("nil clone", func(t *testing.T) {
		var c *EmbeddingConfig
		assert.Nil(t, c.Clone())
	})

	t.Run("deep copy", func(t *testing.T) {
		original := &EmbeddingConfig{
			Dimensions: intPtr(1536),
			Format:     EmbeddingFormatFloat,
		}

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.Equal(t, *original.Dimensions, *clone.Dimensions)
		assert.Equal(t, original.Format, clone.Format)

		// Verify independence
		*clone.Dimensions = 3072
		assert.NotEqual(t, *original.Dimensions, *clone.Dimensions)
	})
}

// --- EmbeddingConfig.ToMap ---

func TestEmbeddingConfig_ToMap(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("nil", func(t *testing.T) {
		var c *EmbeddingConfig
		assert.Nil(t, c.ToMap())
	})

	t.Run("all fields", func(t *testing.T) {
		c := &EmbeddingConfig{
			Dimensions: intPtr(1536),
			Format:     EmbeddingFormatFloat,
		}

		m := c.ToMap()
		assert.Equal(t, 1536, m[ParamKeyDimensions])
		assert.Equal(t, EmbeddingFormatFloat, m[ParamKeyEncodingFormat])
	})
}

// --- StreamingConfig.Validate ---

func TestStreamingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *StreamingConfig
		wantErr bool
		errMsg  string
	}{
		{name: "nil config", config: nil, wantErr: false},
		{name: "disabled no method", config: &StreamingConfig{Enabled: false}, wantErr: false},
		{name: "disabled with method", config: &StreamingConfig{Enabled: false, Method: "invalid"}, wantErr: false},
		{name: "enabled no method", config: &StreamingConfig{Enabled: true}, wantErr: false},
		{name: "enabled sse", config: &StreamingConfig{Enabled: true, Method: StreamMethodSSE}, wantErr: false},
		{name: "enabled websocket", config: &StreamingConfig{Enabled: true, Method: StreamMethodWebSocket}, wantErr: false},
		{name: "enabled invalid method", config: &StreamingConfig{Enabled: true, Method: "grpc"}, wantErr: true, errMsg: ErrMsgStreamInvalidMethod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- StreamingConfig.Clone ---

func TestStreamingConfig_Clone(t *testing.T) {
	t.Run("nil clone", func(t *testing.T) {
		var c *StreamingConfig
		assert.Nil(t, c.Clone())
	})

	t.Run("deep copy", func(t *testing.T) {
		original := &StreamingConfig{
			Enabled: true,
			Method:  StreamMethodSSE,
		}

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.Equal(t, original.Enabled, clone.Enabled)
		assert.Equal(t, original.Method, clone.Method)
	})
}

// --- StreamingConfig.ToMap ---

func TestStreamingConfig_ToMap(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *StreamingConfig
		assert.Nil(t, c.ToMap())
	})

	t.Run("with method", func(t *testing.T) {
		c := &StreamingConfig{
			Enabled: true,
			Method:  StreamMethodSSE,
		}

		m := c.ToMap()
		assert.Equal(t, true, m[ParamKeyEnabled])
		assert.Equal(t, StreamMethodSSE, m[ParamKeyStreamMethod])
	})
}

// --- AsyncConfig.Validate ---

func TestAsyncConfig_Validate(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		config  *AsyncConfig
		wantErr bool
		errMsg  string
	}{
		{name: "nil config", config: nil, wantErr: false},
		{name: "empty config", config: &AsyncConfig{}, wantErr: false},
		{name: "valid enabled", config: &AsyncConfig{Enabled: true, PollIntervalSeconds: floatPtr(1.0), PollTimeoutSeconds: floatPtr(30.0)}, wantErr: false},
		{name: "poll interval zero", config: &AsyncConfig{PollIntervalSeconds: floatPtr(0.0)}, wantErr: true, errMsg: ErrMsgAsyncPollIntervalInvalid},
		{name: "poll interval negative", config: &AsyncConfig{PollIntervalSeconds: floatPtr(-1.0)}, wantErr: true, errMsg: ErrMsgAsyncPollIntervalInvalid},
		{name: "poll timeout zero", config: &AsyncConfig{PollTimeoutSeconds: floatPtr(0.0)}, wantErr: true, errMsg: ErrMsgAsyncPollTimeoutInvalid},
		{name: "poll timeout negative", config: &AsyncConfig{PollTimeoutSeconds: floatPtr(-1.0)}, wantErr: true, errMsg: ErrMsgAsyncPollTimeoutInvalid},
		{name: "timeout less than interval", config: &AsyncConfig{PollIntervalSeconds: floatPtr(10.0), PollTimeoutSeconds: floatPtr(5.0)}, wantErr: true, errMsg: ErrMsgAsyncPollTimeoutTooSmall},
		{name: "timeout equals interval", config: &AsyncConfig{PollIntervalSeconds: floatPtr(5.0), PollTimeoutSeconds: floatPtr(5.0)}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- AsyncConfig.Clone ---

func TestAsyncConfig_Clone(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }

	t.Run("nil clone", func(t *testing.T) {
		var c *AsyncConfig
		assert.Nil(t, c.Clone())
	})

	t.Run("deep copy", func(t *testing.T) {
		original := &AsyncConfig{
			Enabled:             true,
			PollIntervalSeconds: floatPtr(2.0),
			PollTimeoutSeconds:  floatPtr(60.0),
		}

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.Equal(t, original.Enabled, clone.Enabled)
		assert.Equal(t, *original.PollIntervalSeconds, *clone.PollIntervalSeconds)
		assert.Equal(t, *original.PollTimeoutSeconds, *clone.PollTimeoutSeconds)

		// Verify independence
		*clone.PollIntervalSeconds = 5.0
		assert.NotEqual(t, *original.PollIntervalSeconds, *clone.PollIntervalSeconds)
	})
}

// --- AsyncConfig.ToMap ---

func TestAsyncConfig_ToMap(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }

	t.Run("nil", func(t *testing.T) {
		var c *AsyncConfig
		assert.Nil(t, c.ToMap())
	})

	t.Run("all fields", func(t *testing.T) {
		c := &AsyncConfig{
			Enabled:             true,
			PollIntervalSeconds: floatPtr(2.0),
			PollTimeoutSeconds:  floatPtr(60.0),
		}

		m := c.ToMap()
		assert.Equal(t, true, m[ParamKeyEnabled])
		assert.Equal(t, 2.0, m[ParamKeyPollInterval])
		assert.Equal(t, 60.0, m[ParamKeyPollTimeout])
	})
}

// --- Modality validation ---

func TestIsValidModality(t *testing.T) {
	tests := []struct {
		modality string
		valid    bool
	}{
		{ModalityText, true},
		{ModalityImage, true},
		{ModalityAudioSpeech, true},
		{ModalityAudioTranscription, true},
		{ModalityMusic, true},
		{ModalitySoundEffects, true},
		{ModalityEmbedding, true},
		{"", false},
		{"video", false},
		{"3d_model", false},
	}

	for _, tt := range tests {
		t.Run(tt.modality, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidModality(tt.modality))
		})
	}
}

// --- Quality/Style/Format validators ---

func TestIsValidImageQuality(t *testing.T) {
	assert.True(t, isValidImageQuality(ImageQualityStandard))
	assert.True(t, isValidImageQuality(ImageQualityHD))
	assert.True(t, isValidImageQuality(ImageQualityLow))
	assert.True(t, isValidImageQuality(ImageQualityMedium))
	assert.True(t, isValidImageQuality(ImageQualityHigh))
	assert.False(t, isValidImageQuality("ultra"))
	assert.False(t, isValidImageQuality(""))
}

func TestIsValidImageStyle(t *testing.T) {
	assert.True(t, isValidImageStyle(ImageStyleNatural))
	assert.True(t, isValidImageStyle(ImageStyleVivid))
	assert.False(t, isValidImageStyle("abstract"))
	assert.False(t, isValidImageStyle(""))
}

func TestIsValidAudioFormat(t *testing.T) {
	assert.True(t, isValidAudioFormat(AudioFormatMP3))
	assert.True(t, isValidAudioFormat(AudioFormatOpus))
	assert.True(t, isValidAudioFormat(AudioFormatAAC))
	assert.True(t, isValidAudioFormat(AudioFormatFLAC))
	assert.True(t, isValidAudioFormat(AudioFormatWAV))
	assert.True(t, isValidAudioFormat(AudioFormatPCM))
	assert.False(t, isValidAudioFormat("ogg"))
	assert.False(t, isValidAudioFormat(""))
}

func TestIsValidEmbeddingFormat(t *testing.T) {
	assert.True(t, isValidEmbeddingFormat(EmbeddingFormatFloat))
	assert.True(t, isValidEmbeddingFormat(EmbeddingFormatBase64))
	assert.False(t, isValidEmbeddingFormat("binary"))
	assert.False(t, isValidEmbeddingFormat(""))
}

func TestIsValidStreamMethod(t *testing.T) {
	assert.True(t, isValidStreamMethod(StreamMethodSSE))
	assert.True(t, isValidStreamMethod(StreamMethodWebSocket))
	assert.False(t, isValidStreamMethod("grpc"))
	assert.False(t, isValidStreamMethod(""))
}
