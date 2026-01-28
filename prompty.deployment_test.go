package prompty

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLabel(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantErr bool
	}{
		{
			name:    "valid lowercase label",
			label:   "production",
			wantErr: false,
		},
		{
			name:    "valid label with underscore",
			label:   "my_label",
			wantErr: false,
		},
		{
			name:    "valid label with hyphen",
			label:   "my-label",
			wantErr: false,
		},
		{
			name:    "valid label with numbers",
			label:   "label123",
			wantErr: false,
		},
		{
			name:    "empty label",
			label:   "",
			wantErr: true,
		},
		{
			name:    "label starting with number",
			label:   "123label",
			wantErr: true,
		},
		{
			name:    "label starting with underscore",
			label:   "_label",
			wantErr: true,
		},
		{
			name:    "label starting with hyphen",
			label:   "-label",
			wantErr: true,
		},
		{
			name:    "label with uppercase",
			label:   "Production",
			wantErr: true,
		},
		{
			name:    "label with spaces",
			label:   "my label",
			wantErr: true,
		},
		{
			name:    "label too long",
			label:   "this_is_a_very_long_label_name_that_exceeds_the_maximum_allowed_length_of_64_characters",
			wantErr: true,
		},
		{
			name:    "reserved label production",
			label:   LabelProduction,
			wantErr: false,
		},
		{
			name:    "reserved label staging",
			label:   LabelStaging,
			wantErr: false,
		},
		{
			name:    "reserved label canary",
			label:   LabelCanary,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabel(tt.label)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCanTransitionStatus(t *testing.T) {
	tests := []struct {
		name string
		from DeploymentStatus
		to   DeploymentStatus
		want bool
	}{
		// From draft
		{name: "draft to active", from: DeploymentStatusDraft, to: DeploymentStatusActive, want: true},
		{name: "draft to archived", from: DeploymentStatusDraft, to: DeploymentStatusArchived, want: true},
		{name: "draft to deprecated", from: DeploymentStatusDraft, to: DeploymentStatusDeprecated, want: false},
		{name: "draft to draft", from: DeploymentStatusDraft, to: DeploymentStatusDraft, want: false},

		// From active
		{name: "active to deprecated", from: DeploymentStatusActive, to: DeploymentStatusDeprecated, want: true},
		{name: "active to archived", from: DeploymentStatusActive, to: DeploymentStatusArchived, want: true},
		{name: "active to draft", from: DeploymentStatusActive, to: DeploymentStatusDraft, want: false},
		{name: "active to active", from: DeploymentStatusActive, to: DeploymentStatusActive, want: false},

		// From deprecated
		{name: "deprecated to active", from: DeploymentStatusDeprecated, to: DeploymentStatusActive, want: true},
		{name: "deprecated to archived", from: DeploymentStatusDeprecated, to: DeploymentStatusArchived, want: true},
		{name: "deprecated to draft", from: DeploymentStatusDeprecated, to: DeploymentStatusDraft, want: false},
		{name: "deprecated to deprecated", from: DeploymentStatusDeprecated, to: DeploymentStatusDeprecated, want: false},

		// From archived (terminal state)
		{name: "archived to active", from: DeploymentStatusArchived, to: DeploymentStatusActive, want: false},
		{name: "archived to deprecated", from: DeploymentStatusArchived, to: DeploymentStatusDeprecated, want: false},
		{name: "archived to draft", from: DeploymentStatusArchived, to: DeploymentStatusDraft, want: false},
		{name: "archived to archived", from: DeploymentStatusArchived, to: DeploymentStatusArchived, want: false},

		// Invalid status
		{name: "invalid from status", from: DeploymentStatus("invalid"), to: DeploymentStatusActive, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransitionStatus(tt.from, tt.to)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeploymentStatus_IsValid(t *testing.T) {
	tests := []struct {
		status DeploymentStatus
		want   bool
	}{
		{DeploymentStatusDraft, true},
		{DeploymentStatusActive, true},
		{DeploymentStatusDeprecated, true},
		{DeploymentStatusArchived, true},
		{DeploymentStatus("invalid"), false},
		{DeploymentStatus(""), false},
		{DeploymentStatus("ACTIVE"), false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeploymentStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status DeploymentStatus
		want   bool
	}{
		{DeploymentStatusDraft, false},
		{DeploymentStatusActive, false},
		{DeploymentStatusDeprecated, false},
		{DeploymentStatusArchived, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsTerminal()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeploymentStatus_IsUsable(t *testing.T) {
	tests := []struct {
		status DeploymentStatus
		want   bool
	}{
		{DeploymentStatusDraft, false},
		{DeploymentStatusActive, true},
		{DeploymentStatusDeprecated, true},
		{DeploymentStatusArchived, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsUsable()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseDeploymentStatus(t *testing.T) {
	tests := []struct {
		input   string
		want    DeploymentStatus
		wantErr bool
	}{
		{"draft", DeploymentStatusDraft, false},
		{"active", DeploymentStatusActive, false},
		{"deprecated", DeploymentStatusDeprecated, false},
		{"archived", DeploymentStatusArchived, false},
		{"invalid", "", true},
		{"", "", true},
		{"ACTIVE", "", true}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDeploymentStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAllDeploymentStatuses(t *testing.T) {
	statuses := AllDeploymentStatuses()
	assert.Len(t, statuses, 4)
	assert.Contains(t, statuses, DeploymentStatusDraft)
	assert.Contains(t, statuses, DeploymentStatusActive)
	assert.Contains(t, statuses, DeploymentStatusDeprecated)
	assert.Contains(t, statuses, DeploymentStatusArchived)
}

// Storage implementation tests

func TestMemoryStorage_Labels(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()
	defer storage.Close()

	// Create a template
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: "Hello {~prompty.var name=\"name\" /~}",
	}
	err := storage.Save(ctx, tmpl)
	require.NoError(t, err)
	assert.Equal(t, 1, tmpl.Version)

	// Set a label
	err = storage.SetLabel(ctx, "test-template", "production", 1, "user1")
	require.NoError(t, err)

	// Get by label
	got, err := storage.GetByLabel(ctx, "test-template", "production")
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)

	// List labels
	labels, err := storage.ListLabels(ctx, "test-template")
	require.NoError(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "production", labels[0].Label)
	assert.Equal(t, 1, labels[0].Version)

	// Get version labels
	versionLabels, err := storage.GetVersionLabels(ctx, "test-template", 1)
	require.NoError(t, err)
	assert.Contains(t, versionLabels, "production")

	// Reassign label to new version
	tmpl2 := &StoredTemplate{
		Name:   "test-template",
		Source: "Updated: Hello {~prompty.var name=\"name\" /~}",
	}
	err = storage.Save(ctx, tmpl2)
	require.NoError(t, err)
	assert.Equal(t, 2, tmpl2.Version)

	err = storage.SetLabel(ctx, "test-template", "production", 2, "user2")
	require.NoError(t, err)

	got, err = storage.GetByLabel(ctx, "test-template", "production")
	require.NoError(t, err)
	assert.Equal(t, 2, got.Version)

	// Remove label
	err = storage.RemoveLabel(ctx, "test-template", "production")
	require.NoError(t, err)

	_, err = storage.GetByLabel(ctx, "test-template", "production")
	assert.Error(t, err)
}

func TestMemoryStorage_Status(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()
	defer storage.Close()

	// Create a template - should default to active
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: "Hello",
	}
	err := storage.Save(ctx, tmpl)
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusActive, tmpl.Status)

	// Verify stored status
	got, err := storage.Get(ctx, "test-template")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusActive, got.Status)

	// Transition to deprecated
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusDeprecated, "user1")
	require.NoError(t, err)

	got, err = storage.Get(ctx, "test-template")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDeprecated, got.Status)

	// Transition back to active (allowed)
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusActive, "user1")
	require.NoError(t, err)

	// Transition to archived
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusArchived, "user1")
	require.NoError(t, err)

	// Try to transition from archived (should fail)
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusActive, "user1")
	assert.Error(t, err)

	// List by status
	results, err := storage.ListByStatus(ctx, DeploymentStatusArchived, nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "test-template", results[0].Name)
}

func TestMemoryStorage_LabelValidation(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()
	defer storage.Close()

	// Create a template
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: "Hello",
	}
	err := storage.Save(ctx, tmpl)
	require.NoError(t, err)

	// Invalid label - uppercase
	err = storage.SetLabel(ctx, "test-template", "Production", 1, "")
	assert.Error(t, err)

	// Invalid label - empty
	err = storage.SetLabel(ctx, "test-template", "", 1, "")
	assert.Error(t, err)

	// Label for non-existent version
	err = storage.SetLabel(ctx, "test-template", "production", 999, "")
	assert.Error(t, err)

	// Label for non-existent template
	err = storage.SetLabel(ctx, "non-existent", "production", 1, "")
	assert.Error(t, err)
}

func TestMemoryStorage_StatusWithDraft(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()
	defer storage.Close()

	// Create a template with draft status
	tmpl := &StoredTemplate{
		Name:   "draft-template",
		Source: "WIP content",
		Status: DeploymentStatusDraft,
	}
	err := storage.Save(ctx, tmpl)
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDraft, tmpl.Status)

	// Verify stored status
	got, err := storage.Get(ctx, "draft-template")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDraft, got.Status)

	// Draft to active is allowed
	err = storage.SetStatus(ctx, "draft-template", 1, DeploymentStatusActive, "reviewer")
	require.NoError(t, err)

	// Create another draft and try invalid transition
	tmpl2 := &StoredTemplate{
		Name:   "draft-template",
		Source: "WIP content v2",
		Status: DeploymentStatusDraft,
	}
	err = storage.Save(ctx, tmpl2)
	require.NoError(t, err)

	// Draft to deprecated is NOT allowed
	err = storage.SetStatus(ctx, "draft-template", 2, DeploymentStatusDeprecated, "reviewer")
	assert.Error(t, err)
}

func TestMemoryStorage_LabelAssignmentTracking(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()
	defer storage.Close()

	// Create a template
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: "Hello",
	}
	err := storage.Save(ctx, tmpl)
	require.NoError(t, err)

	// Set a label with specific assignedBy
	beforeSet := time.Now()
	err = storage.SetLabel(ctx, "test-template", "production", 1, "deploy-bot")
	require.NoError(t, err)
	afterSet := time.Now()

	// Verify AssignedAt and AssignedBy are tracked
	labels, err := storage.ListLabels(ctx, "test-template")
	require.NoError(t, err)
	require.Len(t, labels, 1)

	assert.Equal(t, "production", labels[0].Label)
	assert.Equal(t, 1, labels[0].Version)
	assert.Equal(t, "deploy-bot", labels[0].AssignedBy)
	assert.True(t, !labels[0].AssignedAt.Before(beforeSet), "AssignedAt should be >= beforeSet")
	assert.True(t, !labels[0].AssignedAt.After(afterSet), "AssignedAt should be <= afterSet")

	// Reassign the label and verify new timestamp
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	beforeReassign := time.Now()
	err = storage.SetLabel(ctx, "test-template", "production", 1, "admin-user")
	require.NoError(t, err)
	afterReassign := time.Now()

	labels, err = storage.ListLabels(ctx, "test-template")
	require.NoError(t, err)
	require.Len(t, labels, 1)

	assert.Equal(t, "admin-user", labels[0].AssignedBy)
	assert.True(t, !labels[0].AssignedAt.Before(beforeReassign), "AssignedAt should be updated on reassignment")
	assert.True(t, !labels[0].AssignedAt.After(afterReassign), "AssignedAt should be <= afterReassign")
}
