package prompty

// CredentialRef declares a credential required for execution.
// It specifies the provider, an optional human-readable label, a reference
// (e.g., "${ENV_VAR}" or "vault://path"), and optional permission scopes.
//
// go-prompty does NOT resolve credentials; it only stores and validates the declarations.
// Orchestration layers are responsible for resolving Ref values at runtime.
type CredentialRef struct {
	// Provider is the LLM provider this credential authenticates against (required).
	Provider string `yaml:"provider" json:"provider"`
	// Label is a human-readable name for the credential.
	Label string `yaml:"label,omitempty" json:"label,omitempty"`
	// Ref is a reference string for credential resolution (e.g., "${OPENAI_API_KEY}", "vault://secrets/openai").
	Ref string `yaml:"ref,omitempty" json:"ref,omitempty"`
	// Scopes lists permission scopes for this credential (e.g., ["images", "embeddings"]).
	Scopes []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

// Validate checks the credential reference for required fields.
func (c *CredentialRef) Validate() error {
	if c == nil {
		return nil
	}
	if c.Provider == "" {
		return NewCredentialValidationError(ErrMsgCredentialMissingProvider, c.Label)
	}
	return nil
}

// Clone creates a deep copy of the CredentialRef.
func (c *CredentialRef) Clone() *CredentialRef {
	if c == nil {
		return nil
	}
	clone := &CredentialRef{
		Provider: c.Provider,
		Label:    c.Label,
		Ref:      c.Ref,
	}
	if c.Scopes != nil {
		clone.Scopes = make([]string, len(c.Scopes))
		copy(clone.Scopes, c.Scopes)
	}
	return clone
}

// GetProvider returns the provider name.
func (c *CredentialRef) GetProvider() string {
	if c == nil {
		return ""
	}
	return c.Provider
}

// GetLabel returns the credential label.
func (c *CredentialRef) GetLabel() string {
	if c == nil {
		return ""
	}
	return c.Label
}

// GetRef returns the credential reference string.
func (c *CredentialRef) GetRef() string {
	if c == nil {
		return ""
	}
	return c.Ref
}

// GetScopes returns the permission scopes.
func (c *CredentialRef) GetScopes() []string {
	if c == nil {
		return nil
	}
	return c.Scopes
}

// HasScopes returns true if any scopes are defined.
func (c *CredentialRef) HasScopes() bool {
	return c != nil && len(c.Scopes) > 0
}
