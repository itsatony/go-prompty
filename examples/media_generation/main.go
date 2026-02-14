// Package main demonstrates media generation configuration with go-prompty v2.5.
//
// This example shows how to configure ExecutionConfig for image generation,
// audio TTS, and embedding generation with provider-specific serialization.
package main

import (
	"fmt"
	"log"

	prompty "github.com/itsatony/go-prompty/v2"
)

func main() {
	imageGenerationExample()
	audioTTSExample()
	embeddingExample()
	streamingExample()
}

func imageGenerationExample() {
	fmt.Println("=== Image Generation (DALL-E 3) ===")

	source := `---
name: image-gen
description: Generate product images
type: prompt
execution:
  modality: image
  provider: openai
  model: dall-e-3
  image:
    size: "1024x1024"
    quality: hd
    style: vivid
    num_images: 2
---
Generate a professional product photo of {~prompty.var name="product" /~}`

	p, err := prompty.Parse([]byte(source))
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	if err := p.Execution.Validate(); err != nil {
		log.Fatalf("validation error: %v", err)
	}

	fmt.Printf("Modality: %s\n", p.Execution.GetModality())
	fmt.Printf("Provider: %s\n", p.Execution.GetProvider())
	fmt.Printf("Image Quality: %s\n", p.Execution.Image.Quality)
	fmt.Printf("Image Size: %s\n", p.Execution.Image.EffectiveSize())

	// Serialize to OpenAI format
	openAI := p.Execution.ToOpenAI()
	fmt.Printf("OpenAI params: model=%v size=%v quality=%v style=%v n=%v\n",
		openAI["model"], openAI[prompty.ParamKeyImageSize], openAI[prompty.ParamKeyImageQuality], openAI[prompty.ParamKeyImageStyle], openAI[prompty.ParamKeyImageN])
	fmt.Println()
}

func audioTTSExample() {
	fmt.Println("=== Audio TTS (OpenAI) ===")

	source := `---
name: narrator
description: Generate audiobook narration
type: prompt
execution:
  modality: audio_speech
  provider: openai
  model: tts-1-hd
  audio:
    voice: alloy
    speed: 1.25
    output_format: mp3
---
{~prompty.var name="text" /~}`

	p, err := prompty.Parse([]byte(source))
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	if err := p.Execution.Validate(); err != nil {
		log.Fatalf("validation error: %v", err)
	}

	fmt.Printf("Modality: %s\n", p.Execution.GetModality())
	fmt.Printf("Voice: %s\n", p.Execution.Audio.Voice)
	fmt.Printf("Speed: %.2f\n", *p.Execution.Audio.Speed)
	fmt.Printf("Format: %s\n", p.Execution.Audio.OutputFormat)

	openAI := p.Execution.ToOpenAI()
	fmt.Printf("OpenAI params: model=%v voice=%v speed=%v response_format=%v\n",
		openAI["model"], openAI[prompty.ParamKeyVoice], openAI[prompty.ParamKeySpeed], openAI[prompty.ParamKeyResponseFormat])
	fmt.Println()
}

func embeddingExample() {
	fmt.Println("=== Embedding (OpenAI) ===")

	source := `---
name: embedder
description: Generate text embeddings
type: prompt
execution:
  modality: embedding
  provider: openai
  model: text-embedding-3-small
  embedding:
    dimensions: 1536
    format: float
---
{~prompty.var name="input" /~}`

	p, err := prompty.Parse([]byte(source))
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	if err := p.Execution.Validate(); err != nil {
		log.Fatalf("validation error: %v", err)
	}

	fmt.Printf("Modality: %s\n", p.Execution.GetModality())
	fmt.Printf("Dimensions: %d\n", *p.Execution.Embedding.Dimensions)
	fmt.Printf("Format: %s\n", p.Execution.Embedding.Format)

	openAI := p.Execution.ToOpenAI()
	fmt.Printf("OpenAI params: model=%v dimensions=%v encoding_format=%v\n",
		openAI["model"], openAI[prompty.ParamKeyDimensions], openAI[prompty.ParamKeyEncodingFormat])
	fmt.Println()
}

func streamingExample() {
	fmt.Println("=== Streaming + Async ===")

	config := &prompty.ExecutionConfig{
		Provider: prompty.ProviderOpenAI,
		Model:    "gpt-4",
		Streaming: &prompty.StreamingConfig{
			Enabled: true,
			Method:  prompty.StreamMethodSSE,
		},
		Async: &prompty.AsyncConfig{
			Enabled:             true,
			PollIntervalSeconds: func() *float64 { v := 2.0; return &v }(),
			PollTimeoutSeconds:  func() *float64 { v := 120.0; return &v }(),
		},
	}

	if err := config.Validate(); err != nil {
		log.Fatalf("validation error: %v", err)
	}

	fmt.Printf("Streaming: enabled=%v method=%s\n", config.Streaming.Enabled, config.Streaming.Method)
	fmt.Printf("Async: enabled=%v interval=%.0fs timeout=%.0fs\n",
		config.Async.Enabled,
		*config.Async.PollIntervalSeconds,
		*config.Async.PollTimeoutSeconds)

	openAI := config.ToOpenAI()
	fmt.Printf("OpenAI stream=%v\n", openAI[prompty.ParamKeyStream])

	// Clone and merge
	clone := config.Clone()
	clone.Streaming.Method = prompty.StreamMethodWebSocket
	fmt.Printf("Clone streaming method: %s (original: %s)\n", clone.Streaming.Method, config.Streaming.Method)
	fmt.Println()
}
