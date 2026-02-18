package prompty

import "sort"

// ExecutionSlot represents a single execution point within an agent — either the
// primary execution or a skill's execution.
type ExecutionSlot struct {
	// Slug identifies this execution slot (agent name for primary, skill slug for skills).
	Slug string `json:"slug"`
	// Execution is the execution config for this slot.
	Execution *ExecutionConfig `json:"execution,omitempty"`
	// Credential is the credential label from the agent's credentials map.
	Credential string `json:"credential,omitempty"`
	// Requirements are the execution requirements for this slot.
	Requirements *ExecutionRequirements `json:"requirements,omitempty"`
}

// ExecutionManifest is a compiled introspection of all provider, model, credential,
// and modality needs for an agent. It is produced by CompileExecutionManifest()
// and enables orchestration layers to pre-validate, route, and schedule execution.
//
// go-prompty generates the manifest from configuration only — no template execution occurs.
type ExecutionManifest struct {
	// Primary is the agent's primary execution slot.
	Primary ExecutionSlot `json:"primary"`
	// Skills maps skill slugs to their execution slots.
	Skills map[string]ExecutionSlot `json:"skills,omitempty"`
	// Providers is a deduplicated, sorted list of all providers used.
	Providers []string `json:"providers"`
	// Credentials is a deduplicated, sorted list of all credential labels referenced.
	Credentials []string `json:"credentials"`
	// Modalities is a deduplicated, sorted list of all modalities across slots.
	Modalities []string `json:"modalities"`
}

// HasProvider returns true if the given provider is used by any execution slot.
func (m *ExecutionManifest) HasProvider(provider string) bool {
	if m == nil {
		return false
	}
	for _, p := range m.Providers {
		if p == provider {
			return true
		}
	}
	return false
}

// RequiresCredential returns true if the given credential label is required.
func (m *ExecutionManifest) RequiresCredential(label string) bool {
	if m == nil {
		return false
	}
	for _, c := range m.Credentials {
		if c == label {
			return true
		}
	}
	return false
}

// SkillsForProvider returns the slugs of all skills that use the given provider.
func (m *ExecutionManifest) SkillsForProvider(provider string) []string {
	if m == nil || m.Skills == nil {
		return nil
	}
	var result []string
	for slug, slot := range m.Skills {
		if slot.Execution != nil && slot.Execution.GetEffectiveProvider() == provider {
			result = append(result, slug)
		}
	}
	sort.Strings(result)
	return result
}

// CompileExecutionManifest builds an ExecutionManifest from the prompt configuration.
// This is a pure metadata operation — no template execution occurs.
// It inspects the primary execution config, all skill refs, and credential declarations
// to produce a complete picture of what providers, credentials, and modalities are needed.
func (p *Prompt) CompileExecutionManifest() (*ExecutionManifest, error) {
	if p == nil {
		return nil, NewManifestError(ErrMsgManifestNilPrompt, nil)
	}

	providerSet := make(map[string]bool)
	credentialSet := make(map[string]bool)
	modalitySet := make(map[string]bool)

	// Primary slot
	primary := ExecutionSlot{
		Slug:         p.Name,
		Execution:    p.Execution,
		Credential:   p.Credential,
		Requirements: p.Requirements,
	}

	// Collect from primary
	if p.Execution != nil {
		if provider := p.Execution.GetEffectiveProvider(); provider != "" {
			providerSet[provider] = true
		}
		if p.Execution.Modality != "" {
			modalitySet[p.Execution.Modality] = true
		}
	}
	if p.Credential != "" {
		credentialSet[p.Credential] = true
	}
	if p.Requirements != nil && p.Requirements.Modality != "" {
		modalitySet[p.Requirements.Modality] = true
	}

	// If no explicit modality, default to text for primary
	if len(modalitySet) == 0 {
		modalitySet[ModalityText] = true
	}

	// Skill slots
	var skillSlots map[string]ExecutionSlot
	if len(p.Skills) > 0 {
		skillSlots = make(map[string]ExecutionSlot, len(p.Skills))
		for i := range p.Skills {
			skill := &p.Skills[i]
			slug := skill.GetSlug()

			slot := ExecutionSlot{
				Slug:         slug,
				Execution:    skill.Execution,
				Credential:   skill.Credential,
				Requirements: skill.Requirements,
			}

			// Collect provider from skill execution
			if skill.Execution != nil {
				if provider := skill.Execution.GetEffectiveProvider(); provider != "" {
					providerSet[provider] = true
				}
				if skill.Execution.Modality != "" {
					modalitySet[skill.Execution.Modality] = true
				}
			}
			if skill.Credential != "" {
				credentialSet[skill.Credential] = true
			}
			if skill.Requirements != nil && skill.Requirements.Modality != "" {
				modalitySet[skill.Requirements.Modality] = true
			}

			skillSlots[slug] = slot
		}
	}

	// Also collect all declared credential labels
	for label := range p.Credentials {
		credentialSet[label] = true
	}

	manifest := &ExecutionManifest{
		Primary:     primary,
		Skills:      skillSlots,
		Providers:   sortedKeys(providerSet),
		Credentials: sortedKeys(credentialSet),
		Modalities:  sortedKeys(modalitySet),
	}

	return manifest, nil
}

// sortedKeys returns the keys of a map as a sorted string slice.
func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
