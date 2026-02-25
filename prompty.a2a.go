package prompty

import (
	"context"
	"encoding/json"
	"strings"
)

// A2AAgentCard represents a Google A2A protocol Agent Card (v0.3).
// An Agent Card describes an agent's capabilities, skills, and communication modes
// for discovery and orchestration on A2A networks.
//
// go-prompty generates Agent Cards from Prompt configuration via CompileAgentCard().
// This is a pure metadata transformation — no template execution or network communication occurs.
type A2AAgentCard struct {
	// Name is the agent's display name (required)
	Name string `json:"name"`
	// Description of the agent's purpose
	Description string `json:"description,omitempty"`
	// URL is the agent's service endpoint (required, provided via options)
	URL string `json:"url"`
	// Version of the agent implementation
	Version string `json:"version,omitempty"`
	// ProtocolVersion is the A2A protocol version (defaults to "0.3.0")
	ProtocolVersion string `json:"protocolVersion"`
	// Provider identifies the organization running the agent
	Provider *A2AProvider `json:"provider,omitempty"`
	// Capabilities advertises what the agent supports
	Capabilities *A2ACapabilities `json:"capabilities,omitempty"`
	// Skills lists the agent's advertised capabilities
	Skills []A2ASkill `json:"skills,omitempty"`
	// DefaultInputModes lists accepted input MIME types (e.g., "text/plain", "application/json")
	DefaultInputModes []string `json:"defaultInputModes,omitempty"`
	// DefaultOutputModes lists produced output MIME types
	DefaultOutputModes []string `json:"defaultOutputModes,omitempty"`
	// SecuritySchemes defines inbound authentication methods.
	// Uses map[string]any because A2A security schemes are defined by the OpenAPI spec
	// and vary widely per scheme type (bearer, apiKey, oauth2, etc.).
	SecuritySchemes map[string]any `json:"securitySchemes,omitempty"`
	// Security references required security schemes
	Security []map[string][]string `json:"security,omitempty"`
	// Metadata contains additional key-value pairs.
	// Uses map[string]any because A2A metadata is an open-ended extension point
	// with no fixed schema — values can be strings, numbers, arrays, or nested objects.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// A2AProvider identifies the organization running an agent.
type A2AProvider struct {
	// Organization is the provider's name
	Organization string `json:"organization"`
	// URL is the provider's website
	URL string `json:"url,omitempty"`
}

// A2ACapabilities advertises what protocol features the agent supports.
type A2ACapabilities struct {
	// Streaming indicates if the agent supports streaming responses
	Streaming bool `json:"streaming,omitempty"`
	// PushNotifications indicates if the agent supports push notifications
	PushNotifications bool `json:"pushNotifications,omitempty"`
}

// A2ASkill represents a capability the agent advertises to other agents.
type A2ASkill struct {
	// ID is the unique identifier for this skill
	ID string `json:"id"`
	// Name is the display name
	Name string `json:"name"`
	// Description explains what the skill does
	Description string `json:"description,omitempty"`
	// Tags for categorization
	Tags []string `json:"tags,omitempty"`
	// InputModes overrides the agent's default input modes for this skill
	InputModes []string `json:"inputModes,omitempty"`
	// OutputModes overrides the agent's default output modes for this skill
	OutputModes []string `json:"outputModes,omitempty"`
}

// A2AAgentCardOptions configures how a Prompt is compiled into an A2A Agent Card.
type A2AAgentCardOptions struct {
	// URL is the agent's service endpoint (required — not derivable from Prompt)
	URL string
	// ProviderOrganization is the organization name for the Agent Card's provider field
	ProviderOrganization string
	// ProviderURL is the organization's website URL
	ProviderURL string
	// Version overrides the agent version (defaults to "1.0.0")
	Version string
	// ProtocolVersion overrides the A2A protocol version (defaults to "0.3.0")
	ProtocolVersion string
	// DefaultInputModes overrides auto-detected input MIME types
	DefaultInputModes []string
	// DefaultOutputModes overrides auto-detected output MIME types
	DefaultOutputModes []string
	// SecuritySchemes defines inbound authentication configuration
	SecuritySchemes map[string]any
	// Security references which security schemes are required
	Security []map[string][]string
	// Capabilities overrides auto-detected capabilities
	Capabilities *A2ACapabilities
	// Resolver resolves skill descriptions from external documents
	Resolver DocumentResolver
}

// CompileAgentCard generates an A2A Agent Card from this Prompt's configuration.
// This is a pure metadata transformation — no template execution occurs.
//
// The URL must be provided via options (it represents the deployment endpoint,
// which is not part of the Prompt definition). Name is taken from the Prompt
// and must not be empty.
//
// Skills are mapped from SkillRef entries. Descriptions are resolved via
// opts.Resolver when available; resolution failures are non-fatal (the skill
// appears with an empty description, following the same pattern as GenerateSkillsCatalog).
//
// Streaming capability is auto-detected from ExecutionConfig.Streaming.Enabled
// unless overridden via opts.Capabilities.
//
// Input modes are inferred from Prompt.Inputs types (string->"text/plain",
// object->"application/json", etc.). Output modes are inferred from the
// Execution/Requirements modality. Both can be overridden via options.
func (p *Prompt) CompileAgentCard(ctx context.Context, opts *A2AAgentCardOptions) (*A2AAgentCard, error) {
	if p == nil {
		return nil, NewA2AError(ErrMsgA2ACardNilPrompt, nil)
	}
	if opts == nil || opts.URL == "" {
		return nil, NewA2AError(ErrMsgA2ACardMissingURL, nil)
	}
	if p.Name == "" {
		return nil, NewA2AError(ErrMsgA2ACardMissingName, nil)
	}

	card := &A2AAgentCard{
		Name:            p.Name,
		Description:     p.Description,
		URL:             opts.URL,
		ProtocolVersion: A2AProtocolVersionDefault,
	}

	// Version
	if opts.Version != "" {
		card.Version = opts.Version
	} else {
		card.Version = A2AVersionDefault
	}

	// Protocol version override
	if opts.ProtocolVersion != "" {
		card.ProtocolVersion = opts.ProtocolVersion
	}

	// Provider
	if opts.ProviderOrganization != "" {
		card.Provider = &A2AProvider{
			Organization: opts.ProviderOrganization,
			URL:          opts.ProviderURL,
		}
	}

	// Capabilities: auto-detect streaming unless overridden
	if opts.Capabilities != nil {
		card.Capabilities = opts.Capabilities
	} else {
		card.Capabilities = a2aAutoDetectCapabilities(p)
	}

	// Skills
	card.Skills = a2aCompileSkills(ctx, p, opts.Resolver)

	// Input modes: override or auto-detect from Inputs
	if len(opts.DefaultInputModes) > 0 {
		card.DefaultInputModes = opts.DefaultInputModes
	} else {
		card.DefaultInputModes = a2aInferInputModes(p)
	}

	// Output modes: override or auto-detect from modality
	if len(opts.DefaultOutputModes) > 0 {
		card.DefaultOutputModes = opts.DefaultOutputModes
	} else {
		card.DefaultOutputModes = a2aInferOutputModes(p)
	}

	// Security
	card.SecuritySchemes = opts.SecuritySchemes
	card.Security = opts.Security

	// Metadata: merge prompt metadata + a2a-prefixed extensions
	card.Metadata = a2aBuildMetadata(p)

	return card, nil
}

// ToJSON serializes the Agent Card to compact JSON.
func (c *A2AAgentCard) ToJSON() ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// ToJSONPretty serializes the Agent Card to indented JSON.
func (c *A2AAgentCard) ToJSONPretty() ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return json.MarshalIndent(c, "", JSONIndentDefault)
}

// --- Internal helpers ---

// a2aAutoDetectCapabilities detects capabilities from the Prompt's ExecutionConfig.
func a2aAutoDetectCapabilities(p *Prompt) *A2ACapabilities {
	caps := &A2ACapabilities{}
	if p.Execution != nil && p.Execution.Streaming != nil && p.Execution.Streaming.Enabled {
		caps.Streaming = true
	}
	return caps
}

// a2aCompileSkills maps SkillRef entries to A2A skills.
func a2aCompileSkills(ctx context.Context, p *Prompt, resolver DocumentResolver) []A2ASkill {
	if len(p.Skills) == 0 {
		return nil
	}

	skills := make([]A2ASkill, 0, len(p.Skills))

	for i := range p.Skills {
		ref := &p.Skills[i]
		slug := ref.GetSlug()

		skill := A2ASkill{
			ID:   slug,
			Name: slug,
		}

		// Resolve description (and optionally name) from inline or external resolver
		if ref.IsInline() && ref.Inline != nil {
			skill.Description = ref.Inline.Description
		} else if resolver != nil {
			resolved, err := resolver.ResolveSkill(ctx, slug)
			if err == nil && resolved != nil {
				skill.Description = resolved.Description
				if resolved.Name != "" {
					skill.Name = resolved.Name
				}
			}
			// Resolution failures are non-fatal
		}

		// Infer output modes from skill's execution modality or requirements
		if ref.Execution != nil && ref.Execution.Modality != "" {
			if mime := modalityToMIME(ref.Execution.Modality); mime != "" {
				skill.OutputModes = []string{mime}
			}
		} else if ref.Requirements != nil && ref.Requirements.Modality != "" {
			if mime := modalityToMIME(ref.Requirements.Modality); mime != "" {
				skill.OutputModes = []string{mime}
			}
		}

		skills = append(skills, skill)
	}

	return skills
}

// a2aInferInputModes infers MIME types from Prompt.Inputs definitions.
func a2aInferInputModes(p *Prompt) []string {
	if len(p.Inputs) == 0 {
		return []string{A2AMIMETextPlain}
	}

	mimeSet := make(map[string]bool)
	for _, def := range p.Inputs {
		if def == nil {
			continue
		}
		mime := inputTypeToMIME(def.Type)
		mimeSet[mime] = true
	}

	if len(mimeSet) == 0 {
		return []string{A2AMIMETextPlain}
	}

	return sortedKeys(mimeSet)
}

// a2aInferOutputModes infers output MIME types from the Prompt's modality configuration.
func a2aInferOutputModes(p *Prompt) []string {
	// Check requirements modality first, then execution modality
	modality := ""
	if p.Requirements != nil && p.Requirements.Modality != "" {
		modality = p.Requirements.Modality
	} else if p.Execution != nil && p.Execution.Modality != "" {
		modality = p.Execution.Modality
	}

	if modality == "" {
		return []string{A2AMIMETextPlain}
	}

	mime := modalityToMIME(modality)
	if mime == "" {
		return []string{A2AMIMETextPlain}
	}
	return []string{mime}
}

// a2aBuildMetadata merges Prompt.Metadata with a2a-prefixed extensions.
func a2aBuildMetadata(p *Prompt) map[string]any {
	var meta map[string]any

	// Start with prompt metadata
	if len(p.Metadata) > 0 {
		meta = make(map[string]any, len(p.Metadata))
		for k, v := range p.Metadata {
			meta[k] = v
		}
	}

	// Merge a2a-prefixed extensions
	for k, v := range p.Extensions {
		if strings.HasPrefix(k, ExtensionPrefixA2A) {
			if meta == nil {
				meta = make(map[string]any)
			}
			meta[k] = v
		}
	}

	return meta
}

// modalityToMIME converts a go-prompty modality constant to an A2A MIME type.
func modalityToMIME(modality string) string {
	switch modality {
	case ModalityText:
		return A2AMIMETextPlain
	case ModalityImage, ModalityImageEdit:
		return A2AMIMEImagePNG
	case ModalityAudioSpeech, ModalityAudioTranscription, ModalityMusic, ModalitySoundEffects:
		return A2AMIMEAudioMPEG
	case ModalityEmbedding:
		return A2AMIMEApplicationJSON
	case ModalityVideo:
		// Video generation APIs return structured JSON metadata (URLs, dimensions, status)
		// rather than raw binary video streams — application/json is the correct wire type.
		return A2AMIMEApplicationJSON
	default:
		return ""
	}
}

// inputTypeToMIME converts a go-prompty InputDef type to an A2A MIME type.
func inputTypeToMIME(inputType string) string {
	switch inputType {
	case SchemaTypeString:
		return A2AMIMETextPlain
	case SchemaTypeObject, SchemaTypeArray:
		return A2AMIMEApplicationJSON
	case SchemaTypeNumber, SchemaTypeBoolean:
		return A2AMIMETextPlain
	default:
		return A2AMIMETextPlain
	}
}
