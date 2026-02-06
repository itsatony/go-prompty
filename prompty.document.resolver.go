package prompty

import (
	"context"
	"strings"
	"sync"
)

// DocumentResolver resolves v2.1 documents (prompts, skills, agents) by slug.
// This is distinct from the tag Resolver interface which handles {~...~} tags.
type DocumentResolver interface {
	// ResolvePrompt resolves a prompt by slug.
	ResolvePrompt(ctx context.Context, slug string) (*Prompt, error)
	// ResolveSkill resolves a skill by slug reference (may include @version).
	ResolveSkill(ctx context.Context, ref string) (*Prompt, error)
	// ResolveAgent resolves an agent by slug.
	ResolveAgent(ctx context.Context, slug string) (*Prompt, error)
}

// NoopDocumentResolver is a DocumentResolver that always returns errors.
// Use this as a default when no resolver is configured.
type NoopDocumentResolver struct{}

// ResolvePrompt always returns an error.
func (r *NoopDocumentResolver) ResolvePrompt(_ context.Context, slug string) (*Prompt, error) {
	return nil, NewRefNotFoundError(slug, RefVersionLatest)
}

// ResolveSkill always returns an error.
func (r *NoopDocumentResolver) ResolveSkill(_ context.Context, ref string) (*Prompt, error) {
	return nil, NewRefNotFoundError(ref, RefVersionLatest)
}

// ResolveAgent always returns an error.
func (r *NoopDocumentResolver) ResolveAgent(_ context.Context, slug string) (*Prompt, error) {
	return nil, NewRefNotFoundError(slug, RefVersionLatest)
}

// MapDocumentResolver is an in-memory DocumentResolver backed by maps.
// Useful for testing and simple use cases. Safe for concurrent access.
type MapDocumentResolver struct {
	mu      sync.RWMutex
	prompts map[string]*Prompt
	skills  map[string]*Prompt
	agents  map[string]*Prompt
}

// NewMapDocumentResolver creates a new MapDocumentResolver.
func NewMapDocumentResolver() *MapDocumentResolver {
	return &MapDocumentResolver{
		prompts: make(map[string]*Prompt),
		skills:  make(map[string]*Prompt),
		agents:  make(map[string]*Prompt),
	}
}

// AddPrompt registers a prompt by slug.
func (r *MapDocumentResolver) AddPrompt(slug string, p *Prompt) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prompts[slug] = p
}

// AddSkill registers a skill by slug.
func (r *MapDocumentResolver) AddSkill(slug string, p *Prompt) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[slug] = p
}

// AddAgent registers an agent by slug.
func (r *MapDocumentResolver) AddAgent(slug string, p *Prompt) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[slug] = p
}

// ResolvePrompt looks up a prompt by slug.
func (r *MapDocumentResolver) ResolvePrompt(_ context.Context, slug string) (*Prompt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.prompts[slug]; ok {
		return p.Clone(), nil
	}
	// Also check skills (skill is the default type)
	if p, ok := r.skills[slug]; ok {
		return p.Clone(), nil
	}
	return nil, NewRefNotFoundError(slug, RefVersionLatest)
}

// ResolveSkill looks up a skill by slug reference.
func (r *MapDocumentResolver) ResolveSkill(_ context.Context, ref string) (*Prompt, error) {
	// Parse slug@version if present
	slug := ref
	if atIdx := len(ref) - 1; atIdx > 0 {
		for i := len(ref) - 1; i > 0; i-- {
			if ref[i] == '@' {
				slug = ref[:i]
				break
			}
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.skills[slug]; ok {
		return p.Clone(), nil
	}
	if p, ok := r.prompts[slug]; ok {
		return p.Clone(), nil
	}
	return nil, NewSkillNotFoundError(slug)
}

// ResolveAgent looks up an agent by slug.
func (r *MapDocumentResolver) ResolveAgent(_ context.Context, slug string) (*Prompt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.agents[slug]; ok {
		return p.Clone(), nil
	}
	return nil, NewRefNotFoundError(slug, RefVersionLatest)
}

// StorageDocumentResolver resolves documents from a TemplateStorage backend.
// It looks up stored templates by slug/name and returns their PromptConfig.
type StorageDocumentResolver struct {
	storage TemplateStorage
}

// NewStorageDocumentResolver creates a resolver backed by the given storage.
func NewStorageDocumentResolver(storage TemplateStorage) *StorageDocumentResolver {
	return &StorageDocumentResolver{storage: storage}
}

// ResolvePrompt looks up a prompt by slug from storage.
func (r *StorageDocumentResolver) ResolvePrompt(ctx context.Context, slug string) (*Prompt, error) {
	return r.resolveByName(ctx, slug)
}

// ResolveSkill looks up a skill by slug reference from storage.
// Supports slug@version format; the version portion is ignored for now
// (storage resolves the latest version by name).
func (r *StorageDocumentResolver) ResolveSkill(ctx context.Context, ref string) (*Prompt, error) {
	slug := ref
	if idx := strings.LastIndex(ref, "@"); idx > 0 {
		slug = ref[:idx]
	}
	return r.resolveByName(ctx, slug)
}

// ResolveAgent looks up an agent by slug from storage.
func (r *StorageDocumentResolver) ResolveAgent(ctx context.Context, slug string) (*Prompt, error) {
	return r.resolveByName(ctx, slug)
}

// resolveByName fetches a stored template by name and returns its PromptConfig.
func (r *StorageDocumentResolver) resolveByName(ctx context.Context, name string) (*Prompt, error) {
	tmpl, err := r.storage.Get(ctx, name)
	if err != nil {
		return nil, NewRefNotFoundError(name, RefVersionLatest)
	}
	if tmpl.PromptConfig != nil {
		return tmpl.PromptConfig.Clone(), nil
	}
	// Template exists but has no PromptConfig — return minimal prompt with body from source
	return &Prompt{
		Name: tmpl.Name,
		Body: tmpl.Source,
	}, nil
}
