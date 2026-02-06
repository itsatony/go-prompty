package prompty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkopeConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *SkopeConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  &SkopeConfig{},
			wantErr: false,
		},
		{
			name: "valid config",
			config: &SkopeConfig{
				Slug:          "my-prompt",
				Visibility:    SkopeVisibilityPublic,
				VersionNumber: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid slug - uppercase",
			config: &SkopeConfig{
				Slug: "MyPrompt",
			},
			wantErr: true,
		},
		{
			name: "invalid slug - starts with number",
			config: &SkopeConfig{
				Slug: "1prompt",
			},
			wantErr: true,
		},
		{
			name: "invalid visibility",
			config: &SkopeConfig{
				Visibility: "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative version number",
			config: &SkopeConfig{
				VersionNumber: -1,
			},
			wantErr: true,
		},
		{
			name: "valid visibility - public",
			config: &SkopeConfig{
				Visibility: SkopeVisibilityPublic,
			},
			wantErr: false,
		},
		{
			name: "valid visibility - private",
			config: &SkopeConfig{
				Visibility: SkopeVisibilityPrivate,
			},
			wantErr: false,
		},
		{
			name: "valid visibility - team",
			config: &SkopeConfig{
				Visibility: SkopeVisibilityTeam,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSkopeConfig_Clone(t *testing.T) {
	now := time.Now()
	original := &SkopeConfig{
		Slug:          "test-prompt",
		ForkedFrom:    "original-prompt",
		CreatedAt:     &now,
		CreatedBy:     "user1",
		UpdatedAt:     &now,
		UpdatedBy:     "user2",
		VersionNumber: 5,
		Visibility:    SkopeVisibilityPublic,
		Projects:      []string{"project1", "project2"},
		References:    []string{"ref1", "ref2"},
	}

	clone := original.Clone()

	// Verify equality
	assert.Equal(t, original.Slug, clone.Slug)
	assert.Equal(t, original.ForkedFrom, clone.ForkedFrom)
	assert.Equal(t, original.CreatedBy, clone.CreatedBy)
	assert.Equal(t, original.VersionNumber, clone.VersionNumber)
	assert.Equal(t, original.Visibility, clone.Visibility)
	assert.Equal(t, original.Projects, clone.Projects)
	assert.Equal(t, original.References, clone.References)

	// Verify deep copy
	clone.Projects[0] = "modified"
	assert.NotEqual(t, original.Projects[0], clone.Projects[0])

	clone.References[0] = "modified"
	assert.NotEqual(t, original.References[0], clone.References[0])
}

func TestSkopeConfig_Getters(t *testing.T) {
	now := time.Now()
	config := &SkopeConfig{
		Slug:          "test-prompt",
		ForkedFrom:    "original-prompt",
		CreatedAt:     &now,
		CreatedBy:     "user1",
		UpdatedAt:     &now,
		UpdatedBy:     "user2",
		VersionNumber: 5,
		Visibility:    SkopeVisibilityPublic,
		Projects:      []string{"project1"},
		References:    []string{"ref1"},
	}

	assert.Equal(t, "test-prompt", config.GetSlug())
	assert.Equal(t, "original-prompt", config.GetForkedFrom())
	assert.NotNil(t, config.GetCreatedAt())
	assert.Equal(t, "user1", config.GetCreatedBy())
	assert.NotNil(t, config.GetUpdatedAt())
	assert.Equal(t, "user2", config.GetUpdatedBy())
	assert.Equal(t, 5, config.GetVersionNumber())
	assert.Equal(t, SkopeVisibilityPublic, config.GetVisibility())
	assert.NotNil(t, config.GetProjects())
	assert.NotNil(t, config.GetReferences())

	// Test nil config
	var nilConfig *SkopeConfig
	assert.Empty(t, nilConfig.GetSlug())
	assert.Empty(t, nilConfig.GetForkedFrom())
	assert.Nil(t, nilConfig.GetCreatedAt())
	assert.Empty(t, nilConfig.GetCreatedBy())
	assert.Equal(t, 0, nilConfig.GetVersionNumber())
}

func TestSkopeConfig_VisibilityHelpers(t *testing.T) {
	tests := []struct {
		name       string
		visibility string
		isPublic   bool
		isPrivate  bool
		isTeam     bool
	}{
		{
			name:       "public",
			visibility: SkopeVisibilityPublic,
			isPublic:   true,
		},
		{
			name:       "private",
			visibility: SkopeVisibilityPrivate,
			isPrivate:  true,
		},
		{
			name:       "team",
			visibility: SkopeVisibilityTeam,
			isTeam:     true,
		},
		{
			name:       "empty",
			visibility: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SkopeConfig{Visibility: tt.visibility}
			assert.Equal(t, tt.isPublic, config.IsPublic())
			assert.Equal(t, tt.isPrivate, config.IsPrivate())
			assert.Equal(t, tt.isTeam, config.IsTeam())
		})
	}
}

func TestSkopeConfig_HasHelpers(t *testing.T) {
	config := &SkopeConfig{
		ForkedFrom: "original",
		Projects:   []string{"project1"},
		References: []string{"ref1"},
	}

	assert.True(t, config.HasForkedFrom())
	assert.True(t, config.HasProjects())
	assert.True(t, config.HasReferences())

	emptyConfig := &SkopeConfig{}
	assert.False(t, emptyConfig.HasForkedFrom())
	assert.False(t, emptyConfig.HasProjects())
	assert.False(t, emptyConfig.HasReferences())
}

func TestSkopeConfig_SetTimestamps(t *testing.T) {
	config := &SkopeConfig{}

	config.SetCreatedNow("user1")
	assert.NotNil(t, config.CreatedAt)
	assert.Equal(t, "user1", config.CreatedBy)

	config.SetUpdatedNow("user2")
	assert.NotNil(t, config.UpdatedAt)
	assert.Equal(t, "user2", config.UpdatedBy)
}

func TestSkopeConfig_AddReference(t *testing.T) {
	config := &SkopeConfig{}

	config.AddReference("ref1")
	assert.Equal(t, []string{"ref1"}, config.References)

	config.AddReference("ref2")
	assert.Equal(t, []string{"ref1", "ref2"}, config.References)

	// Adding duplicate should not add
	config.AddReference("ref1")
	assert.Equal(t, []string{"ref1", "ref2"}, config.References)

	// Adding empty should not add
	config.AddReference("")
	assert.Equal(t, []string{"ref1", "ref2"}, config.References)
}

func TestSkopeConfig_AddProject(t *testing.T) {
	config := &SkopeConfig{}

	config.AddProject("project1")
	assert.Equal(t, []string{"project1"}, config.Projects)

	config.AddProject("project2")
	assert.Equal(t, []string{"project1", "project2"}, config.Projects)

	// Adding duplicate should not add
	config.AddProject("project1")
	assert.Equal(t, []string{"project1", "project2"}, config.Projects)

	// Adding empty should not add
	config.AddProject("")
	assert.Equal(t, []string{"project1", "project2"}, config.Projects)
}

func TestSkopeConfig_JSONAndYAML(t *testing.T) {
	config := &SkopeConfig{
		Slug:       "test-prompt",
		Visibility: SkopeVisibilityPublic,
	}

	jsonStr, err := config.JSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "test-prompt")
	assert.Contains(t, jsonStr, "public")

	yamlStr, err := config.YAML()
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "test-prompt")
	assert.Contains(t, yamlStr, "public")
}
