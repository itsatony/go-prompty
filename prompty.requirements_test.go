package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionRequirements_Validate(t *testing.T) {
	t.Run("nil is valid", func(t *testing.T) {
		var r *ExecutionRequirements
		assert.NoError(t, r.Validate())
	})

	t.Run("empty is valid", func(t *testing.T) {
		r := &ExecutionRequirements{}
		assert.NoError(t, r.Validate())
	})

	t.Run("valid full config", func(t *testing.T) {
		r := &ExecutionRequirements{
			Modality:           ModalityImage,
			ProviderBinding:    ProviderBindingRequired,
			Capabilities:       []string{"image_generation"},
			EstimatedCost:      "$0.04",
			EstimatedLatencyMs: 8000,
		}
		assert.NoError(t, r.Validate())
	})

	t.Run("all provider bindings valid", func(t *testing.T) {
		for _, binding := range []string{ProviderBindingRequired, ProviderBindingPreferred, ProviderBindingAny} {
			r := &ExecutionRequirements{ProviderBinding: binding}
			assert.NoError(t, r.Validate(), "binding %q should be valid", binding)
		}
	})

	t.Run("invalid provider binding", func(t *testing.T) {
		r := &ExecutionRequirements{ProviderBinding: "mandatory"}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgInvalidProviderBinding)
	})

	t.Run("negative latency", func(t *testing.T) {
		r := &ExecutionRequirements{EstimatedLatencyMs: -100}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEstimatedLatencyNegative)
	})

	t.Run("zero latency is valid", func(t *testing.T) {
		r := &ExecutionRequirements{EstimatedLatencyMs: 0}
		assert.NoError(t, r.Validate())
	})

	t.Run("invalid modality", func(t *testing.T) {
		r := &ExecutionRequirements{Modality: "hologram"}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgInvalidModality)
	})

	t.Run("valid modalities", func(t *testing.T) {
		for _, m := range []string{ModalityText, ModalityImage, ModalityEmbedding, ModalityAudioSpeech} {
			r := &ExecutionRequirements{Modality: m}
			assert.NoError(t, r.Validate(), "modality %q should be valid", m)
		}
	})
}

func TestExecutionRequirements_Clone(t *testing.T) {
	t.Run("nil clone", func(t *testing.T) {
		var r *ExecutionRequirements
		assert.Nil(t, r.Clone())
	})

	t.Run("deep copy", func(t *testing.T) {
		r := &ExecutionRequirements{
			Modality:           ModalityImage,
			ProviderBinding:    ProviderBindingRequired,
			Capabilities:       []string{"image_generation", "function_calling"},
			EstimatedCost:      "$0.04",
			EstimatedLatencyMs: 8000,
		}
		clone := r.Clone()
		require.NotNil(t, clone)
		assert.Equal(t, r.Modality, clone.Modality)
		assert.Equal(t, r.ProviderBinding, clone.ProviderBinding)
		assert.Equal(t, r.Capabilities, clone.Capabilities)
		assert.Equal(t, r.EstimatedCost, clone.EstimatedCost)
		assert.Equal(t, r.EstimatedLatencyMs, clone.EstimatedLatencyMs)

		// Verify isolation
		clone.Capabilities[0] = "modified"
		assert.Equal(t, "image_generation", r.Capabilities[0])
	})
}

func TestExecutionRequirements_Getters(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *ExecutionRequirements
		assert.Equal(t, "", r.GetModality())
		assert.False(t, r.HasModality())
		assert.Equal(t, "", r.GetProviderBinding())
		assert.False(t, r.HasProviderBinding())
		assert.Nil(t, r.GetCapabilities())
		assert.False(t, r.HasCapabilities())
		assert.Equal(t, "", r.GetEstimatedCost())
		assert.False(t, r.HasEstimatedCost())
		assert.Equal(t, 0, r.GetEstimatedLatencyMs())
		assert.False(t, r.HasEstimatedLatencyMs())
	})

	t.Run("populated", func(t *testing.T) {
		r := &ExecutionRequirements{
			Modality:           ModalityEmbedding,
			ProviderBinding:    ProviderBindingPreferred,
			Capabilities:       []string{"embeddings"},
			EstimatedCost:      "$0.001",
			EstimatedLatencyMs: 200,
		}
		assert.Equal(t, ModalityEmbedding, r.GetModality())
		assert.True(t, r.HasModality())
		assert.Equal(t, ProviderBindingPreferred, r.GetProviderBinding())
		assert.True(t, r.HasProviderBinding())
		assert.Equal(t, []string{"embeddings"}, r.GetCapabilities())
		assert.True(t, r.HasCapabilities())
		assert.Equal(t, "$0.001", r.GetEstimatedCost())
		assert.True(t, r.HasEstimatedCost())
		assert.Equal(t, 200, r.GetEstimatedLatencyMs())
		assert.True(t, r.HasEstimatedLatencyMs())
	})
}

func TestIsValidProviderBinding(t *testing.T) {
	assert.True(t, isValidProviderBinding(ProviderBindingRequired))
	assert.True(t, isValidProviderBinding(ProviderBindingPreferred))
	assert.True(t, isValidProviderBinding(ProviderBindingAny))
	assert.False(t, isValidProviderBinding(""))
	assert.False(t, isValidProviderBinding("mandatory"))
	assert.False(t, isValidProviderBinding("optional"))
}
