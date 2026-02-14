package prompty

import (
	"encoding/json"
	"time"

	"gopkg.in/yaml.v3"
)

// SkopeConfig represents the v2.0 namespaced Skope platform configuration.
// This contains metadata specific to the Skope prompt management platform.
type SkopeConfig struct {
	// Slug is the unique identifier for this prompt in Skope
	Slug string `yaml:"slug,omitempty" json:"slug,omitempty"`

	// ForkedFrom indicates the slug of the prompt this was forked from
	ForkedFrom string `yaml:"forked_from,omitempty" json:"forked_from,omitempty"`

	// Timestamps and authorship
	CreatedAt *time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	CreatedBy string     `yaml:"created_by,omitempty" json:"created_by,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
	UpdatedBy string     `yaml:"updated_by,omitempty" json:"updated_by,omitempty"`

	// Version tracking
	VersionNumber int `yaml:"version_number,omitempty" json:"version_number,omitempty"`

	// Access control
	Visibility string `yaml:"visibility,omitempty" json:"visibility,omitempty"` // "public", "private", "team"

	// Organization
	Projects []string `yaml:"projects,omitempty" json:"projects,omitempty"`

	// Dependencies (slugs of referenced prompts)
	References []string `yaml:"references,omitempty" json:"references,omitempty"`
}

// Validate checks the skope config for consistency.
func (s *SkopeConfig) Validate() error {
	if s == nil {
		return nil
	}

	// Validate slug format if provided
	if s.Slug != "" && !slugRegex.MatchString(s.Slug) {
		return NewPromptValidationError(ErrMsgInvalidSkopeSlug, s.Slug)
	}

	// Validate visibility if provided
	if s.Visibility != "" {
		switch s.Visibility {
		case SkopeVisibilityPublic, SkopeVisibilityPrivate, SkopeVisibilityTeam:
			// Valid
		default:
			return NewPromptValidationError(ErrMsgInvalidVisibility, s.Visibility)
		}
	}

	// Validate version number if provided
	if s.VersionNumber < 0 {
		return NewPromptValidationError(ErrMsgVersionNumberNegative, "")
	}

	return nil
}

// Clone creates a deep copy of the skope config.
func (s *SkopeConfig) Clone() *SkopeConfig {
	if s == nil {
		return nil
	}

	clone := &SkopeConfig{
		Slug:          s.Slug,
		ForkedFrom:    s.ForkedFrom,
		CreatedBy:     s.CreatedBy,
		UpdatedBy:     s.UpdatedBy,
		VersionNumber: s.VersionNumber,
		Visibility:    s.Visibility,
	}

	if s.CreatedAt != nil {
		t := *s.CreatedAt
		clone.CreatedAt = &t
	}
	if s.UpdatedAt != nil {
		t := *s.UpdatedAt
		clone.UpdatedAt = &t
	}

	if s.Projects != nil {
		clone.Projects = make([]string, len(s.Projects))
		copy(clone.Projects, s.Projects)
	}

	if s.References != nil {
		clone.References = make([]string, len(s.References))
		copy(clone.References, s.References)
	}

	return clone
}

// GetSlug returns the slug or empty string.
func (s *SkopeConfig) GetSlug() string {
	if s == nil {
		return ""
	}
	return s.Slug
}

// GetForkedFrom returns the forked_from slug or empty string.
func (s *SkopeConfig) GetForkedFrom() string {
	if s == nil {
		return ""
	}
	return s.ForkedFrom
}

// GetCreatedAt returns the created timestamp or nil.
func (s *SkopeConfig) GetCreatedAt() *time.Time {
	if s == nil {
		return nil
	}
	return s.CreatedAt
}

// GetCreatedBy returns the creator or empty string.
func (s *SkopeConfig) GetCreatedBy() string {
	if s == nil {
		return ""
	}
	return s.CreatedBy
}

// GetUpdatedAt returns the updated timestamp or nil.
func (s *SkopeConfig) GetUpdatedAt() *time.Time {
	if s == nil {
		return nil
	}
	return s.UpdatedAt
}

// GetUpdatedBy returns the updater or empty string.
func (s *SkopeConfig) GetUpdatedBy() string {
	if s == nil {
		return ""
	}
	return s.UpdatedBy
}

// GetVersionNumber returns the version number or 0.
func (s *SkopeConfig) GetVersionNumber() int {
	if s == nil {
		return 0
	}
	return s.VersionNumber
}

// GetVisibility returns the visibility or empty string.
func (s *SkopeConfig) GetVisibility() string {
	if s == nil {
		return ""
	}
	return s.Visibility
}

// GetProjects returns the projects list or nil.
func (s *SkopeConfig) GetProjects() []string {
	if s == nil {
		return nil
	}
	return s.Projects
}

// GetReferences returns the references list or nil.
func (s *SkopeConfig) GetReferences() []string {
	if s == nil {
		return nil
	}
	return s.References
}

// IsPublic returns true if visibility is public.
func (s *SkopeConfig) IsPublic() bool {
	return s != nil && s.Visibility == SkopeVisibilityPublic
}

// IsPrivate returns true if visibility is private.
func (s *SkopeConfig) IsPrivate() bool {
	return s != nil && s.Visibility == SkopeVisibilityPrivate
}

// IsTeam returns true if visibility is team.
func (s *SkopeConfig) IsTeam() bool {
	return s != nil && s.Visibility == SkopeVisibilityTeam
}

// HasForkedFrom returns true if this prompt was forked.
func (s *SkopeConfig) HasForkedFrom() bool {
	return s != nil && s.ForkedFrom != ""
}

// HasReferences returns true if this prompt has references.
func (s *SkopeConfig) HasReferences() bool {
	return s != nil && len(s.References) > 0
}

// HasProjects returns true if this prompt has projects.
func (s *SkopeConfig) HasProjects() bool {
	return s != nil && len(s.Projects) > 0
}

// JSON returns the JSON representation of the skope config.
func (s *SkopeConfig) JSON() (string, error) {
	if s == nil {
		return "", nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// YAML returns the YAML representation of the skope config.
func (s *SkopeConfig) YAML() (string, error) {
	if s == nil {
		return "", nil
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SetCreatedNow sets CreatedAt and CreatedBy with the current time and user.
func (s *SkopeConfig) SetCreatedNow(user string) {
	if s == nil {
		return
	}
	now := time.Now().UTC()
	s.CreatedAt = &now
	s.CreatedBy = user
}

// SetUpdatedNow sets UpdatedAt and UpdatedBy with the current time and user.
func (s *SkopeConfig) SetUpdatedNow(user string) {
	if s == nil {
		return
	}
	now := time.Now().UTC()
	s.UpdatedAt = &now
	s.UpdatedBy = user
}

// AddReference adds a reference slug if not already present.
func (s *SkopeConfig) AddReference(slug string) {
	if s == nil || slug == "" {
		return
	}
	for _, ref := range s.References {
		if ref == slug {
			return
		}
	}
	s.References = append(s.References, slug)
}

// AddProject adds a project if not already present.
func (s *SkopeConfig) AddProject(project string) {
	if s == nil || project == "" {
		return
	}
	for _, p := range s.Projects {
		if p == project {
			return
		}
	}
	s.Projects = append(s.Projects, project)
}
